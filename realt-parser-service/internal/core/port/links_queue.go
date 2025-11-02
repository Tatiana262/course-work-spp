package port

import (
	"context"
	"realt-parser-service/internal/core/domain"
)

// PropertyLinkQueuePort определяет контракт для отправки ссылок в очередь
type LinksQueuePort interface {
	Enqueue(ctx context.Context, link domain.PropertyLink) error
}