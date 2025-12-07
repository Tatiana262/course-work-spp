package usecases_port

import (
	"context"
	"task-service/internal/core/domain"

	"github.com/google/uuid"
)

type CreateTaskUseCasePort interface {
	Execute(ctx context.Context, name, taskType string, userID uuid.UUID) (*domain.Task, error)
}