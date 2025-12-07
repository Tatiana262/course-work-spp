package usecases_port

import (
	"context"
	"storage-service/internal/core/domain"

	"github.com/google/uuid"
)


type SavePropertyPort interface {
	Save(ctx context.Context, record domain.RealEstateRecord) error
	BatchSave(ctx context.Context, records []domain.RealEstateRecord, taskID uuid.UUID) error
}