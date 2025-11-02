package port

import (
	"context"
	"realt-parser-service/internal/core/domain"
)

// ProcessedPropertyQueuePort определяет контракт для отправки обоаботанных объектов в очередь
type ProcessedPropertyQueuePort interface {
	Enqueue(ctx context.Context, link domain.RealEstateRecord) error
}