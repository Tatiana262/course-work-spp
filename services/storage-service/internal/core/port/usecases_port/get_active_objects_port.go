package usecases_port

import (
	"context"
	"storage-service/internal/core/domain"

)

type GetActiveObjectsUseCase interface {
	FindActiveIDsForActualization(ctx context.Context, category string, limit int) ([]domain.PropertyBasicInfo, error)
}