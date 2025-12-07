package usecases_port

import (
	"context"
	"task-service/internal/core/domain"

	"github.com/google/uuid"
)

type UpdateTaskStatusUseCasePort interface {
	Execute(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, summary *domain.ResultSummary) (*domain.Task, error)
}