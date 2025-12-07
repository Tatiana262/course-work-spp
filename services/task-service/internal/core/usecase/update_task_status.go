package usecase

import (
	"context"
	"task-service/internal/contextkeys"
	"task-service/internal/core/domain"
	"task-service/internal/core/port"
	"time"

	"github.com/google/uuid"
)

type UpdateTaskStatusUseCase struct {
	repo     port.TaskRepositoryPort
	notifier port.NotifierPort
}

func NewUpdateTaskStatusUseCase(repo port.TaskRepositoryPort, notifier port.NotifierPort) *UpdateTaskStatusUseCase {
	return &UpdateTaskStatusUseCase{
		repo:    repo,
		notifier:    notifier,
	}
}

func (uc *UpdateTaskStatusUseCase) Execute(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, summary *domain.ResultSummary) (*domain.Task, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{"use_case": "UpdateTaskStatus", "task_id": taskID.String(), "new_status": status})

	ucLogger.Info("Use case started", nil)
	
	task, err := uc.repo.FindByID(ctx, taskID)
	if err != nil {
		ucLogger.Error("Repository failed to find task", err, nil)
		return nil, err
	}

	task.Status = status
	now := time.Now().UTC()
	if status == domain.StatusRunning && task.StartedAt == nil {
		task.StartedAt = &now
	}
	if status == domain.StatusCompleted || status == domain.StatusFailed {
		task.FinishedAt = &now
	}
	if summary != nil && *summary != nil {
		task.ResultSummary = *summary
	}

	if err := uc.repo.Update(ctx, task); err != nil {
		ucLogger.Error("Repository failed to update task", err, nil)
		return nil, err
	}

	ucLogger.Info("Task status updated, notifying clients", nil)
	// Отправляем уведомление об обновлении
	uc.notifier.Notify(ctx, port.TaskEvent{Type: "task_updated", Data: *task})
	ucLogger.Info("Use case finished successfully", nil)

	return task, nil
}