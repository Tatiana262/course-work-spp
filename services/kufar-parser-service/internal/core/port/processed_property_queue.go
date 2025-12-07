package port

import (
	"context"
	"kufar-parser-service/internal/core/domain"

	"github.com/google/uuid"
)

// PropertyLinkQueuePort определяет контракт для отправки ссылок в очередь.
type ProcessedPropertyQueuePort interface {
	Enqueue(ctx context.Context, link domain.RealEstateRecord, taskID uuid.UUID) error
}