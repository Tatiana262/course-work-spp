package usecase

import (
	"context"
	"task-service/internal/contextkeys"
	"task-service/internal/core/domain"
	"task-service/internal/core/port"

	"github.com/google/uuid"
)

type GetTaskByIdUseCase struct {
	repo port.TaskRepositoryPort
}

func NewGetTaskByIdUseCase(repo port.TaskRepositoryPort) *GetTaskByIdUseCase {
	return &GetTaskByIdUseCase{
		repo: repo,
	}
}

func (uc *GetTaskByIdUseCase) Execute(ctx context.Context, taskId uuid.UUID) (*domain.Task, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{"use_case": "GetTaskById", "task_id": taskId.String()})
	
	ucLogger.Info("Use case started", nil)
	
	task, err := uc.repo.FindByID(ctx, taskId)
	if err != nil {
		ucLogger.Error("Repository failed to find task", err, nil)
		return nil, err
	}

	ucLogger.Info("Use case finished successfully", nil)
	return task, nil
}