package usecase

import (
	"context"
	"task-service/internal/contextkeys"
	"task-service/internal/core/domain"
	"task-service/internal/core/port"

	"github.com/google/uuid"
)

type GetTasksListUseCase struct {
	repo port.TaskRepositoryPort
}

func NewGetTasksListUseCase(repo port.TaskRepositoryPort) *GetTasksListUseCase {
	return &GetTasksListUseCase{
		repo: repo,
	}
}

func (uc *GetTasksListUseCase) Execute(ctx context.Context, createdByUserID uuid.UUID, limit, offset int) ([]domain.Task, int64, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{"use_case": "GetTasksList", "user_id": createdByUserID.String(), "limit": limit, "offset": offset})

	ucLogger.Info("Use case started", nil)

	
	tasks, count, err := uc.repo.FindAll(ctx, createdByUserID, limit, offset)
	if err != nil {
		ucLogger.Error("Repository failed to find tasks", err, nil)
		return nil, 0, err
	}

	ucLogger.Info("Use case finished successfully", port.Fields{"found_on_page": len(tasks), "total_count": count})
	return tasks, count, nil
}