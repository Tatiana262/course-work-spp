package usecases_port

import (
	"context"

	"github.com/google/uuid"
)

type CompleteTaskUseCasePort interface {
	Execute(ctx context.Context, taskID uuid.UUID, expectedCount int) error
}