package usecases_port

import (
	"context"
	"task-service/internal/core/domain"

	"github.com/google/uuid"
)

type GetTaskByIdUseCasePort interface {
	Execute(ctx context.Context, taskId uuid.UUID) (*domain.Task, error)
}