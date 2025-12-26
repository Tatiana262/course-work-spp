package port

import (
	"actualization-service/internal/core/domain"
	"context"
)

type LinksSearchQueuePort interface {
	PublishTask(ctx context.Context, task domain.FindNewLinksTask) error
}
