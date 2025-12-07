package port

import (
	"actualization-service/internal/core/domain"
	"context"
)

type LinkSearchQueuePort interface {
	PublishTask(ctx context.Context, task domain.FindNewLinksTask) error
}
