package port

import (
	"context"
	"github.com/google/uuid"
)

type UserTaskServicePort interface {
	CreateTask(ctx context.Context, name, taskType string, userID uuid.UUID) (uuid.UUID, error)
	UpdateTaskStatus(ctx context.Context, taskID uuid.UUID, status string) error
}