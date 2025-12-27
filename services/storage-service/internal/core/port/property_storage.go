package port

import (
	"context"
	"storage-service/internal/core/domain"

	"github.com/google/uuid"
)


type PropertyStoragePort interface {
	Save(ctx context.Context, record domain.RealEstateRecord) error
	BatchSave(ctx context.Context, records []domain.RealEstateRecord) (*domain.BatchSaveStats, error)

	GetActiveIDsForActualization(ctx context.Context, category string, limit int) ([]domain.PropertyBasicInfo, error)
	GetArchivedIDsForActualization(ctx context.Context, category string, limit int) ([]domain.PropertyBasicInfo, error)
	GetObjectsByIDForActualization(ctx context.Context, masterObjectID string) ([]domain.PropertyBasicInfo, error)
	GetActualizationStats(ctx context.Context) ([]domain.StatsByCategory, error)
	
	FindWithFilters(ctx context.Context, filters domain.FindObjectsFilters, limit, offset int) (*domain.PaginatedResult, error)
    GetPropertyDetails(ctx context.Context, propertyID uuid.UUID) (*domain.PropertyDetailsView, error)
	FindBestByMasterIDs(ctx context.Context, masterIDs []string) ([]domain.GeneralPropertyInfo, error)
}