package port

import (
	"actualization-service/internal/core/domain"
	"context"
)

// TaskManagerPort - контракт для адаптера.
type TaskResultsPort interface {
	PublishCompletionCommand(ctx context.Context, cmd domain.TaskCompletionCommand) error
}
