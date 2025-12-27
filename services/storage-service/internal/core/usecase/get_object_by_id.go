package usecase

import (
	"context"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"
)

type GetObjectByIDUseCase struct {
    storage port.PropertyStoragePort
}

func NewGetObjectsByIDUseCase(storage port.PropertyStoragePort) *GetObjectByIDUseCase {
    return &GetObjectByIDUseCase{storage: storage}
}


func (uc *GetObjectByIDUseCase) FindObjectsByIDForActualization(ctx context.Context, id string) ([]domain.PropertyBasicInfo, error) {
	logger := contextkeys.LoggerFromContext(ctx)
    ucLogger := logger.WithFields(port.Fields{
        "use_case": "FindObjectByIDForActualization",
        "id": id,
    })

    ucLogger.Info("Use case started", nil)

	result, err := uc.storage.GetObjectsByIDForActualization(ctx, id)
    if err != nil {
        ucLogger.Error("Storage returned an error", err, nil)
        return nil, err
    }

    ucLogger.Info("Use case finished successfully", nil)
    
    return result, nil

}