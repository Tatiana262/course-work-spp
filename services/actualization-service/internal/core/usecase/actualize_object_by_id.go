package usecase

import (
	"actualization-service/internal/constants"
	"actualization-service/internal/contextkeys"
	"actualization-service/internal/core/domain"
	"actualization-service/internal/core/port" // Используем порты
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
)

type ActualizeObjectByIdUseCase struct {
	storage     port.StoragePort
	taskQueue   port.ParsingTaskQueuePort
	taskService port.UserTaskServicePort
	taskResults port.TaskResultsPort
}

func NewActualizeObjectByIdUseCase(storage port.StoragePort,
	taskQueue port.ParsingTaskQueuePort,
	taskService port.UserTaskServicePort,
	taskResults port.TaskResultsPort) *ActualizeObjectByIdUseCase {
	return &ActualizeObjectByIdUseCase{
		storage:     storage,
		taskQueue:   taskQueue,
		taskService: taskService,
		taskResults: taskResults,
	}
}

// Execute - основной метод
func (uc *ActualizeObjectByIdUseCase) Execute(ctx context.Context, userID uuid.UUID, id string) (uuid.UUID, error) {

	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case": "ActualizeObjectById",
		"user_id":  userID,
	})

	traceID := contextkeys.TraceIDFromContext(ctx)
	backgroundCtx := context.Background()
	backgroundCtx = contextkeys.ContextWithLogger(backgroundCtx, logger)
	backgroundCtx = contextkeys.ContextWithTraceID(backgroundCtx, traceID)

	// Шаг 1: Создаем задачу в task-service
	taskName := fmt.Sprintf("Актуализация объекта (id: %s)", id)
	taskID, err := uc.taskService.CreateTask(ctx, taskName, "ACTUALIZE_BY_ID", userID)
	if err != nil {
		ucLogger.Error("Could not create user task", err, nil)
		return uuid.Nil, fmt.Errorf("could not create task: %w", err)
	}

	ucLogger.Info("User task created successfully, starting background processing", port.Fields{"task_id": taskID.String()})

	// Шаг 2: Запускаем основную логику в фоновой горутине, чтобы немедленно вернуть ответ.
	go uc.runInBackground(backgroundCtx, taskID, id)

	// Шаг 3: Немедленно возвращаем ID задачи.
	return taskID, nil

}

// runInBackground - приватный метод для выполнения долгой работы.
func (uc *ActualizeObjectByIdUseCase) runInBackground(ctx context.Context, taskID uuid.UUID, id string) {

	logger := contextkeys.LoggerFromContext(ctx)
	taskLogger := logger.WithFields(port.Fields{
		"use_case":  "ActualizeObjectById.background",
		"task_id":   taskID.String(),
		"object_id": id,
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
	taskLogger.Info("Fetching object from storage", nil)
	object, err := uc.storage.GetObjectByID(ctx, id)
	if err != nil {
		taskLogger.Error("Failed to get object from storage", err, nil)
		uc.taskService.UpdateTaskStatus(ctx, taskID, "failed")
		return
	}

	if object == nil {
		taskLogger.Info("No objects to actualize. Sending completion command.", nil)
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
		taskLogger.Info("Found object to actualize", nil)
	}

	object.TaskID = taskID

	log.Println(object.Source, object.AdID, object.Link, object.TaskID)

	task := domain.ActualizationTask{
		Task:     *object,
		Priority: domain.ACTUALIZE_OBJECT,
	}
	if object.Source == domain.KUFAR_SOURCE {
		task.RoutingKey = constants.RoutingKeyLinkTasksKufar
	}
	if object.Source == domain.REALT_SOURCE {
		task.RoutingKey = constants.RoutingKeyLinkTasksRealt
	}
	if err := uc.taskQueue.PublishTask(ctx, task); err != nil {
		taskLogger.Error("Failed to publish actualization object sub-task", err, port.Fields{"link": object.Link})
		uc.taskService.UpdateTaskStatus(ctx, taskID, "failed")
	}

	taskLogger.Info("Sub-task for object dispatched. Sending completion command for user task", nil)
	completionCmd := domain.TaskCompletionCommand{
		TaskID:               taskID,
		ExpectedResultsCount: 1,
	}
	if err := uc.taskResults.PublishCompletionCommand(ctx, completionCmd); err != nil {
		taskLogger.Error("Failed to publish completion command", err, nil)
		uc.taskService.UpdateTaskStatus(ctx, taskID, "failed")
	}

}
