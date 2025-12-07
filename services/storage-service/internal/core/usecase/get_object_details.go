package usecase

import (
	"context"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"

	"github.com/google/uuid"
)

type GetObjectDetailsUseCase struct {
    storage port.PropertyStoragePort
}

func NewGetObjectDetailsUseCase(storage port.PropertyStoragePort) *GetObjectDetailsUseCase {
    return &GetObjectDetailsUseCase{storage: storage}
}

func (uc *GetObjectDetailsUseCase) Execute(ctx context.Context, objectID uuid.UUID) (*domain.PropertyDetailsView, error) {
    logger := contextkeys.LoggerFromContext(ctx)
    ucLogger := logger.WithFields(port.Fields{
        "use_case": "FindObjectByIDForActualization",
        "object_id": objectID.String(),
    })

    ucLogger.Info("Use case started", nil)

    result, err := uc.storage.GetPropertyDetails(ctx, objectID)
    if err != nil {
        ucLogger.Error("Storage returned an error", err, nil)
        return nil, err
    }

    ucLogger.Info("Use case finished successfully", nil)
    
    return result, nil
}