package usecase

import (
	"context"
	"task-service/internal/contextkeys"
	"task-service/internal/core/domain"
	"task-service/internal/core/port"

	"github.com/google/uuid"
)

type CreateTaskUseCase struct {
	repo     port.TaskRepositoryPort
	notifier port.NotifierPort
}

func NewCreateTaskUseCase(repo port.TaskRepositoryPort, notifier port.NotifierPort) *CreateTaskUseCase {
	return &CreateTaskUseCase{
		repo:    repo,
		notifier:    notifier,
	}
}

func (uc *CreateTaskUseCase) Execute(ctx context.Context, name, taskType string, userID uuid.UUID, params... any) (*domain.Task, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case": "CreateTask",
		"task_name": name,
		"task_type": taskType,
		"user_id":  userID.String(),
	})

	ucLogger.Info("Use case started", nil)
	
	task := domain.NewTask(name, taskType, userID)
	if len(params) > 0 {
		task.ResultSummary["id"] = params[0]
	}

	if err := uc.repo.Create(ctx, task); err != nil {
		ucLogger.Error("Repository failed to create task", err, nil)
		return nil, err
	}

	ucLogger = ucLogger.WithFields(port.Fields{"task_id": task.ID.String()}) // Обогащаем ID после создания
	ucLogger.Debug("Task created successfully, notifying clients", nil)

	// Отправляем уведомление о создании задачи
	uc.notifier.Notify(ctx, port.TaskEvent{Type: "task_created", Data: *task})

	ucLogger.Info("Use case finished successfully", nil)
	return task, nil
}