package usecases_port

import (
	"context"
	"storage-service/internal/core/domain"

)


type GetArchivedObjectsUseCase interface {
	FindArchivedIDsForActualization(ctx context.Context, category string, limit int) ([]domain.PropertyBasicInfo, error)
}