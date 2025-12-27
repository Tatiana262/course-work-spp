package usecase

import (
	"context"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"
)

type GetArchivedObjectsUseCase struct {
    storage port.PropertyStoragePort
}

func NewGetArchivedObjectsUseCase(storage port.PropertyStoragePort) *GetArchivedObjectsUseCase {
    return &GetArchivedObjectsUseCase{storage: storage}
}


func (uc *GetArchivedObjectsUseCase) FindArchivedIDsForActualization(ctx context.Context, category string, limit int) ([]domain.PropertyBasicInfo, error) {

    logger := contextkeys.LoggerFromContext(ctx)
    ucLogger := logger.WithFields(port.Fields{
        "use_case": "FindArchivedIDsForActualization",
        "category": category,
        "limit":    limit,
    })

    ucLogger.Info("Use case started", nil)

    result, err := uc.storage.GetArchivedIDsForActualization(ctx, category, limit)
    if err != nil {
        ucLogger.Error("Storage returned an error", err, nil)
        return nil, err
    }

    ucLogger.Info("Use case finished successfully", port.Fields{"found_count": len(result)})
    
    return result, nil
}