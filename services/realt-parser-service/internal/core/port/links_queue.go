package port

import (
	"context"
	"realt-parser-service/internal/core/domain"

	"github.com/google/uuid"
)

// PropertyLinkQueuePort определяет контракт для отправки ссылок в очередь.
type LinksQueuePort interface {
	Enqueue(ctx context.Context, link domain.PropertyLink, taskID uuid.UUID) error
}