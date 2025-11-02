package port

import (
	"context"
	"storage-service/internal/core/domain"
)

// PropertyStoragePort определяет контракт для сохранения
// обработанного объекта недвижимости в постоянное хранилище
type PropertyStoragePort interface {
	Save(ctx context.Context, record domain.RealEstateRecord) error
	BatchSave(ctx context.Context, records []domain.RealEstateRecord) error 

	GetActiveIDsForActualization(ctx context.Context, limit, offset int) ([]domain.PropertyBasicInfo, error)
	GetArchivedIDsForActualization(ctx context.Context, limit, offset int) ([]domain.PropertyBasicInfo, error)
	GetObjectByIDForActualization(ctx context.Context, id string) (*domain.PropertyBasicInfo, error)
	
}