package usecase

import (
	"actualization-service/internal/contextkeys"
	"actualization-service/internal/core/domain"
	"actualization-service/internal/core/port"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ActualizeArchivedObjectsUseCase struct {
	storage     port.StoragePort
	taskQueue   port.LinksQueuePort
	taskService port.UserTaskServicePort
	taskResults port.TaskResultsPort
}

func NewActualizeArchivedObjectsUseCase(storage port.StoragePort,
	taskQueue port.LinksQueuePort,
	taskService port.UserTaskServicePort,
	taskResults port.TaskResultsPort) *ActualizeArchivedObjectsUseCase {
	return &ActualizeArchivedObjectsUseCase{
		storage:     storage,
		taskQueue:   taskQueue,
		taskService: taskService,
		taskResults: taskResults,
	}
}

// Execute - основной метод
func (uc *ActualizeArchivedObjectsUseCase) Execute(ctx context.Context, userID uuid.UUID, category *string, limit int) (uuid.UUID, error) {

	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case": "ActualizeArchivedObjects",
		"user_id":  userID,
	})

	traceID := contextkeys.TraceIDFromContext(ctx)
	backgroundCtx := context.Background()
	backgroundCtx = contextkeys.ContextWithLogger(backgroundCtx, logger)
	backgroundCtx = contextkeys.ContextWithTraceID(backgroundCtx, traceID)

	//  Определяем имя и тип задачи 
	taskName := ""
	if category != nil && *category != "" {
		taskName = fmt.Sprintf("Актуализация %d архивных объектов (Категория: %s)", limit, *category)
	} else {
		taskName = fmt.Sprintf("Массовая актуализация архивных объектов (лимит: %d на категорию)", limit)
	}

	taskID, err := uc.taskService.CreateTask(ctx, taskName, "ACTUALIZE_ARCHIVED", userID)
	if err != nil {
		ucLogger.Error("Could not create user task", err, nil)
		return uuid.Nil, fmt.Errorf("could not create task: %w", err)
	}

	ucLogger.Info("User task created successfully, starting background processing", port.Fields{"task_id": taskID.String()})

	// Запускаем основную логику в фоновой горутине, чтобы немедленно вернуть ответ
	go uc.runInBackground(backgroundCtx, taskID, category, limit)

	// возвращаем ID задачи
	return taskID, nil
}

// runInBackground - приватный метод для выполнения фоновой работы
func (uc *ActualizeArchivedObjectsUseCase) runInBackground(ctx context.Context, taskID uuid.UUID, category *string, limit int) {

	logger := contextkeys.LoggerFromContext(ctx)
	taskLogger := logger.WithFields(port.Fields{
		"use_case": "ActualizeArchivedObjects.background",
		"task_id":  taskID.String(),
		"category": *category,
	})

	// Обновляем статус задачи на "running"
	if err := uc.taskService.UpdateTaskStatus(ctx, taskID, "running"); err != nil {
		taskLogger.Error("Failed to update task status to 'running'", err, nil)
		uc.taskService.UpdateTaskStatus(ctx, taskID, "failed")
		return
	}

	var categoriesToProcess []string
	if category != nil && *category != "" {
		// 1. Задана одна конкретная категория
		categoriesToProcess = []string{*category}
		taskLogger.Debug("Starting single-category actualization", port.Fields{"category": *category})
	} else {
		// 2. Актуализация всех категорий
		taskLogger.Debug("Starting multi-category actualization", nil)
		
        // Получаем список всех категорий от storage-service
		categoryDict, err := uc.storage.GetCategories(ctx)
		if err != nil {
			taskLogger.Error("Failed to get categories from storage", err, nil)
			uc.taskService.UpdateTaskStatus(ctx, taskID, "failed")
			return
		}

		for _, item := range categoryDict {
			categoriesToProcess = append(categoriesToProcess, item.SystemName)
		}
		taskLogger.Info("Found categories to process", port.Fields{"categories": categoriesToProcess})
	}

	totalTasksToDispatch := 0
	var allObjects []domain.PropertyInfo

	// Собираем объекты из всех категорий
	for _, cat := range categoriesToProcess {
		objects, err := uc.storage.GetArchivedObjects(ctx, cat, limit)
		if err != nil {
			taskLogger.Error("Failed to get archived objects for category", err, port.Fields{"category": cat})
			uc.taskService.UpdateTaskStatus(ctx, taskID, "failed")
			return
		}
		allObjects = append(allObjects, objects...)
	}

	totalTasksToDispatch = len(allObjects)
	
	if totalTasksToDispatch == 0 {
		taskLogger.Info("No archived objects to actualize. Sending completion command.", nil)
		completionCmd := domain.TaskCompletionCommand{
			TaskID:               taskID,
			Results: map[string]int{
				"expected_results_count": 0,
			},
			// ExpectedResultsCount: 0,
		}
		if err := uc.taskResults.PublishCompletionCommand(ctx, completionCmd); err != nil {
			taskLogger.Error("Failed to publish zero-count completion command", err, nil)
		}
		uc.taskService.UpdateTaskStatus(ctx, taskID, "completed")
		return
	} else {
		taskLogger.Info("Total objects to actualize across all categories", port.Fields{"count": totalTasksToDispatch})
	}

	for _, obj := range allObjects {
		obj.TaskID = taskID

		task := domain.ActualizationTask{
			Task:     obj,
			Priority: domain.ACTUALIZE_ARCHIVED,
			Source: obj.Source,
		}
		
		if err := uc.taskQueue.PublishTask(ctx, task); err != nil {
			taskLogger.Error("Failed to publish actualization archived sub-task", err, port.Fields{"link": obj.Link})
			uc.taskService.UpdateTaskStatus(ctx, taskID, "failed")
		}
	}

	completionCmd := domain.TaskCompletionCommand{
		TaskID:               taskID,
		Results: map[string]int{
			"expected_results_count": totalTasksToDispatch,
		},
		// ExpectedResultsCount: totalTasksToDispatch,
	}
	if err := uc.taskResults.PublishCompletionCommand(ctx, completionCmd); err != nil {
		taskLogger.Error("Failed to publish completion command", err, nil)
		uc.taskService.UpdateTaskStatus(ctx, taskID, "failed")
	}

}
