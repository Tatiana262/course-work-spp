package usecase

import (
	"actualization-service/internal/constants"
	"actualization-service/internal/contextkeys"
	"actualization-service/internal/core/domain"
	"actualization-service/internal/core/port" // Используем порты
	"context"
	"fmt"

	// "log"

	"github.com/google/uuid"
)

type ActualizeArchivedObjectsUseCase struct {
	storage     port.StoragePort
	taskQueue   port.ParsingTaskQueuePort
	taskService port.UserTaskServicePort
	taskResults port.TaskResultsPort
}

func NewActualizeArchivedObjectsUseCase(storage port.StoragePort,
	taskQueue port.ParsingTaskQueuePort,
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
func (uc *ActualizeArchivedObjectsUseCase) Execute(ctx context.Context, userID uuid.UUID, category string, limit int) (uuid.UUID, error) {

	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case": "ActualizeArchivedObjects",
		"user_id":  userID,
	})

	traceID := contextkeys.TraceIDFromContext(ctx)
	backgroundCtx := context.Background()
	backgroundCtx = contextkeys.ContextWithLogger(backgroundCtx, logger)
	backgroundCtx = contextkeys.ContextWithTraceID(backgroundCtx, traceID)

	// Шаг 1: Создаем задачу в task-service
	taskName := fmt.Sprintf("Актуализация %d архивных объектов (Категория: %s)", limit, category)
	taskID, err := uc.taskService.CreateTask(ctx, taskName, "ACTUALIZE_ARCHIVED", userID)
	if err != nil {
		ucLogger.Error("Could not create user task", err, nil)
		return uuid.Nil, fmt.Errorf("could not create task: %w", err)
	}

	ucLogger.Info("User task created successfully, starting background processing", port.Fields{"task_id": taskID.String()})

	// Шаг 2: Запускаем основную логику в фоновой горутине, чтобы немедленно вернуть ответ.
	go uc.runInBackground(backgroundCtx, taskID, category, limit)

	// Шаг 3: Немедленно возвращаем ID задачи.
	return taskID, nil
}

// runInBackground - приватный метод для выполнения долгой работы.
func (uc *ActualizeArchivedObjectsUseCase) runInBackground(ctx context.Context, taskID uuid.UUID, category string, limit int) {

	logger := contextkeys.LoggerFromContext(ctx)
	taskLogger := logger.WithFields(port.Fields{
		"use_case": "ActualizeArchivedObjects.background",
		"task_id":  taskID.String(),
		"category": category,
	})

	// Шаг 3.1: Обновляем статус задачи на "running"
	if err := uc.taskService.UpdateTaskStatus(ctx, taskID, "running"); err != nil {
		taskLogger.Error("Failed to update task status to 'running'", err, nil)
		// Можно обновить статус на "failed" здесь же
		uc.taskService.UpdateTaskStatus(ctx, taskID, "failed")
		return
	}

	// Шаг 3.2: Выполняем старую логику
	// 1. Получаем список объектов от storage-service
	taskLogger.Info("Fetching archived objects from storage", port.Fields{"limit": limit})
	objects, err := uc.storage.GetArchivedObjects(ctx, category, limit)
	if err != nil {
		taskLogger.Error("Failed to get archived objects from storage", err, nil)
		uc.taskService.UpdateTaskStatus(ctx, taskID, "failed")
		return
	}

	totalTasksToDispatch := len(objects)

	if totalTasksToDispatch == 0 {
		taskLogger.Info("No archived objects to actualize. Sending completion command.", nil)
		completionCmd := domain.TaskCompletionCommand{
			TaskID:               taskID,
			ExpectedResultsCount: 0,
		}
		if err := uc.taskResults.PublishCompletionCommand(ctx, completionCmd); err != nil {
			taskLogger.Error("Failed to publish zero-count completion command", err, nil)
		}
		uc.taskService.UpdateTaskStatus(ctx, taskID, "completed")
		return
	} else {
		taskLogger.Info("Found archived objects to actualize", port.Fields{"count": len(objects)})
	}

	for _, obj := range objects {
		obj.TaskID = taskID

		task := domain.ActualizationTask{
			Task:     obj,
			Priority: domain.ACTUALIZE_ARCHIVED,
		}
		if obj.Source == domain.KUFAR_SOURCE {
			task.RoutingKey = constants.RoutingKeyLinkTasksKufar
		}
		if obj.Source == domain.REALT_SOURCE {
			task.RoutingKey = constants.RoutingKeyLinkTasksRealt
		}
		if err := uc.taskQueue.PublishTask(ctx, task); err != nil {
			taskLogger.Error("Failed to publish actualization archived sub-task", err, port.Fields{"link": obj.Link})
			uc.taskService.UpdateTaskStatus(ctx, taskID, "failed")
		}
	}

	taskLogger.Info("All sub-tasks for archived objects dispatched. Sending completion command for user task", port.Fields{"dispatched_count": totalTasksToDispatch})
	completionCmd := domain.TaskCompletionCommand{
		TaskID:               taskID,
		ExpectedResultsCount: totalTasksToDispatch,
	}
	if err := uc.taskResults.PublishCompletionCommand(ctx, completionCmd); err != nil {
		taskLogger.Error("Failed to publish completion command", err, nil)
		uc.taskService.UpdateTaskStatus(ctx, taskID, "failed")
	}

}
