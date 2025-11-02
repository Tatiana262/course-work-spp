package port

import (
	"context"
	"storage-service/internal/core/domain"

)

type GetActiveObjectsUseCase interface {
	FindActiveIDsForActualization(ctx context.Context, limit, offset int) ([]domain.PropertyBasicInfo, error)
}

type GetArchivedObjectsUseCase interface {
	FindArchivedIDsForActualization(ctx context.Context, limit, offset int) ([]domain.PropertyBasicInfo, error)
}

type GetObjectByIDUseCase interface {
	FindObjectByIDForActualization(ctx context.Context, id string) (*domain.PropertyBasicInfo, error)
}