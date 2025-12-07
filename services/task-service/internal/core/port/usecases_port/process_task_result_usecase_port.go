package usecases_port

import (
	"context"
	"github.com/google/uuid"
)

type ProcessTaskResultUseCasePort interface {
	Execute(ctx context.Context, taskID uuid.UUID, results map[string]int) error
}