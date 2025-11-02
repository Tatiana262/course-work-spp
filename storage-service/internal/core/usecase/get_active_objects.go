package usecase

import (
    "context"
    "storage-service/internal/core/domain"
    "storage-service/internal/core/port"
)

type GetActiveObjectsUseCase struct {
    storage port.PropertyStoragePort
}

func NewGetActiveObjectsUseCase(storage port.PropertyStoragePort) *GetActiveObjectsUseCase {
    return &GetActiveObjectsUseCase{storage: storage}
}

// FindIDsForActualization - метод для получения ID для актуализации
func (uc *GetActiveObjectsUseCase) FindActiveIDsForActualization(ctx context.Context, limit, offset int) ([]domain.PropertyBasicInfo, error) {
    return uc.storage.GetActiveIDsForActualization(ctx, limit, offset)
}