package usecases_port

import (
	"context"
	"storage-service/internal/core/domain"

)

type GetObjectByIDUseCase interface {
	FindObjectsByIDForActualization(ctx context.Context, id string) ([]domain.PropertyBasicInfo, error)
}