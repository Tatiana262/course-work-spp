package port

import (
	"actualization-service/internal/core/domain"
	"context"
)

type TaskResultsPort interface {
	PublishCompletionCommand(ctx context.Context, cmd domain.TaskCompletionCommand) error
}
