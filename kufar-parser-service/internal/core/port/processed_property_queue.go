package port

import (
	"context"
	"kufar-parser-service/internal/core/domain"
)

// ProcessedPropertyQueuePort определяет контракт для отправки обработанных объектов в очередь
type ProcessedPropertyQueuePort interface {
	Enqueue(ctx context.Context, link domain.RealEstateRecord) error
}