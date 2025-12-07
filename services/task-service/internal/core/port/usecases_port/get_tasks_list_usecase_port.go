package usecases_port

import (
	"context"
	"task-service/internal/core/domain"

	"github.com/google/uuid"
)

type GetTasksListUseCasePort interface {
	Execute(ctx context.Context, createdByUserID uuid.UUID, limit, offset int) ([]domain.Task, int64, error)
}