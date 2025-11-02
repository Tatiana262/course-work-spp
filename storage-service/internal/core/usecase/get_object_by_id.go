package usecase

import (
	"context"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"

)

type GetObjectByIDUseCase struct {
    storage port.PropertyStoragePort
}

func NewGetObjectByIDUseCase(storage port.PropertyStoragePort) *GetObjectByIDUseCase {
    return &GetObjectByIDUseCase{storage: storage}
}

func (uc *GetObjectByIDUseCase) FindObjectByIDForActualization(ctx context.Context, id string) (*domain.PropertyBasicInfo, error) {
    return uc.storage.GetObjectByIDForActualization(ctx, id)
}