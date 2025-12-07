package usecase

import (
	"actualization-service/internal/constants"
	"actualization-service/internal/contextkeys"
	"actualization-service/internal/core/domain"
	"actualization-service/internal/core/port" // Используем порты
	"context"
	"fmt"

	// "log"
	"strings"

	"github.com/google/uuid"
)

type FindNewObjectsUseCase struct {
	taskQueue   port.LinkSearchQueuePort
	taskService port.UserTaskServicePort
	taskResults port.TaskResultsPort
}

func NewFindNewObjectsUseCase(
	taskQueue port.LinkSearchQueuePort,
	taskService port.UserTaskServicePort,
	taskResults port.TaskResultsPort) *FindNewObjectsUseCase {
	return &FindNewObjectsUseCase{
		taskQueue:   taskQueue,
		taskService: taskService,
		taskResults: taskResults,
	}
}

// Execute - основной метод
func (uc *FindNewObjectsUseCase) Execute(ctx context.Context, userID uuid.UUID, categories []string, regions []string) (uuid.UUID, error) {

	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case": "FindNewObjects",
		"user_id":  userID,
	})

	traceID := contextkeys.TraceIDFromContext(ctx)
	backgroundCtx := context.Background()
	backgroundCtx = contextkeys.ContextWithLogger(backgroundCtx, logger)
	backgroundCtx = contextkeys.ContextWithTraceID(backgroundCtx, traceID)

	// Шаг 1: Создаем задачу в task-service
	taskName := fmt.Sprintf("Поиск новых объектов (Категории: %v, Регионы: %v)", categories, regions)
	taskID, err := uc.taskService.CreateTask(ctx, taskName, "FIND_NEW", userID)
	if err != nil {
		ucLogger.Error("Could not create user task", err, nil)
		return uuid.Nil, fmt.Errorf("could not create task: %w", err)
	}

	ucLogger.Info("User task created successfully, starting background processing", port.Fields{"task_id": taskID.String()})

	// Шаг 2: Запускаем основную логику в фоновой горутине, чтобы немедленно вернуть ответ.
	go uc.runInBackground(backgroundCtx, taskID, categories, regions)

	return taskID, nil
}

// runInBackground - приватный метод для выполнения долгой работы.
func (uc *FindNewObjectsUseCase) runInBackground(ctx context.Context, taskID uuid.UUID, categories []string, regions []string) {

	logger := contextkeys.LoggerFromContext(ctx)
	taskLogger := logger.WithFields(port.Fields{
		"use_case":   "FindNewObjects.background",
		"task_id":    taskID.String(),
		"categories": strings.Join(categories, ", "),
		"regions":    strings.Join(regions, ", "),
	})

	// Шаг 3.1: Обновляем статус задачи на "running"
	if err := uc.taskService.UpdateTaskStatus(ctx, taskID, "running"); err != nil {
		taskLogger.Error("Failed to update task status to 'running'", err, nil)
		uc.taskService.UpdateTaskStatus(ctx, taskID, "failed")
		return
	}

	// Шаг 3.2: Выполняем старую логику
	// 1. Получаем список объектов от storage-service
	allTasks := uc.generateAllTasks(categories, regions, taskID)

	totalTasksToDispatch := len(allTasks)

	if totalTasksToDispatch == 0 {
		taskLogger.Info("No new object search sub-tasks for user task. Sending completion command.", nil)

		completionCmd := domain.TaskCompletionCommand{
			TaskID:               taskID,
			ExpectedResultsCount: 0,
		}
		if err := uc.taskResults.PublishCompletionCommand(ctx, completionCmd); err != nil {
			taskLogger.Error("Failed to publish zero-count completion command", err, nil)
		}
		uc.taskService.UpdateTaskStatus(ctx, taskID, "completed")
		return
	}

	for _, task := range allTasks {
		if err := uc.taskQueue.PublishTask(ctx, task); err != nil {
			uc.taskService.UpdateTaskStatus(ctx, taskID, "failed")

			taskLogger.Error("Failed to publish task", err, port.Fields{"region_name": task.Task.Region, "category_name": task.Task.Category})
		}
	}

	taskLogger.Info("All sub-tasks for search new objects dispatched. Sending completion command for user task", port.Fields{"dispatched_count": totalTasksToDispatch})

	completionCmd := domain.TaskCompletionCommand{
		TaskID:               taskID,
		ExpectedResultsCount: totalTasksToDispatch,
	}
	if err := uc.taskResults.PublishCompletionCommand(ctx, completionCmd); err != nil {
		taskLogger.Error("Failed to publish completion command", err, nil)
		uc.taskService.UpdateTaskStatus(ctx, taskID, "failed")
	}
}

func (uc *FindNewObjectsUseCase) generateAllTasks(categories []string, regions []string, taskID uuid.UUID) []domain.FindNewLinksTask {

	if len(categories) == 0 {
		categories = []string{"all-categories"}
	}

	if len(regions) == 0 {
		regions = []string{"all-regions"}
	}

	routingKeys := []string{
		constants.RoutingKeySearchTasksRealt,
		constants.RoutingKeySearchTasksKufar,
	}

	var searchTasks []domain.FindNewLinksTask

	for _, region := range regions {
		for _, category := range categories {
			for _, routingKey := range routingKeys {
				task := domain.FindNewLinksTask{
					Task: domain.TaskInfo{
						Category: category,
						Region:   region,
						TaskID:   taskID,
					},
					RoutingKey: routingKey,
				}

				searchTasks = append(searchTasks, task)
			}
		}
	}

	return searchTasks
}
