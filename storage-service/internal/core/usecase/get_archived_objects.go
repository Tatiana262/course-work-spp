package usecase

import (
    "context"
    "storage-service/internal/core/domain"
    "storage-service/internal/core/port"
)

type GetArchivedObjectsUseCase struct {
    storage port.PropertyStoragePort
}

func NewGetArchivedObjectsUseCase(storage port.PropertyStoragePort) *GetArchivedObjectsUseCase {
    return &GetArchivedObjectsUseCase{storage: storage}
}

func (uc *GetArchivedObjectsUseCase) FindArchivedIDsForActualization(ctx context.Context, limit, offset int) ([]domain.PropertyBasicInfo, error) {
    return uc.storage.GetArchivedIDsForActualization(ctx, limit, offset)
}