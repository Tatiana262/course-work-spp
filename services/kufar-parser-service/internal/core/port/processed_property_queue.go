package port

import (
	"context"
	"kufar-parser-service/internal/core/domain"

	"github.com/google/uuid"
)


type ProcessedPropertyQueuePort interface {
	Enqueue(ctx context.Context, link domain.RealEstateRecord, taskID uuid.UUID) error
}