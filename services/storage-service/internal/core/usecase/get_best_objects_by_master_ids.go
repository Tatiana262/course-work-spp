package usecase

import (
	"context"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"
)

type GetBestObjectsByMasterIDsUseCase struct {
    storage port.PropertyStoragePort
}

func NewGetBestObjectsByMasterIDsUseCase(storage port.PropertyStoragePort) *GetBestObjectsByMasterIDsUseCase {
    return &GetBestObjectsByMasterIDsUseCase{storage: storage}
}


func (uc *GetBestObjectsByMasterIDsUseCase) Execute(ctx context.Context, masterIDs []string) ([]domain.GeneralPropertyInfo, error) {
	logger := contextkeys.LoggerFromContext(ctx)
    ucLogger := logger.WithFields(port.Fields{
        "use_case": "GetBestObjectsByMasterIDs",
        "master_ids_amount": len(masterIDs),
    })

    ucLogger.Info("Use case started", nil)

	result, err := uc.storage.FindBestByMasterIDs(ctx, masterIDs)
    if err != nil {
        ucLogger.Error("Storage returned an error", err, nil)
        return nil, err
    }

    ucLogger.Info("Use case finished successfully", port.Fields{"found_count": len(result)})
    
    return result, nil
}