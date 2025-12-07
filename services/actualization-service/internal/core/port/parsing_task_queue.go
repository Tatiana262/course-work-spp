package port

import (
	"actualization-service/internal/core/domain"
	"context"
)

type ParsingTaskQueuePort interface {
	PublishTask(ctx context.Context, task domain.ActualizationTask) error
}
