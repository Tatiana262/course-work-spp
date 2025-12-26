package usecases_port

import (
	"context"
	"task-service/internal/core/domain"

	"github.com/google/uuid"
)

type UpdateTaskStatusUseCasePort interface {
	Execute(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus) (*domain.Task, error)
}