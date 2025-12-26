package usecase

import (
	"context"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"
)

type GetActualizationStatsUseCase struct {
	storage port.PropertyStoragePort
}

func NewGetActualizationStatsUseCase(storage port.PropertyStoragePort) *GetActualizationStatsUseCase {
    return &GetActualizationStatsUseCase{storage: storage}
}

func(uc *GetActualizationStatsUseCase) Execute(ctx context.Context) ([]domain.StatsByCategory, error) {
	logger := contextkeys.LoggerFromContext(ctx)
    ucLogger := logger.WithFields(port.Fields{
        "use_case": "GetActualizationStats",
    })

    ucLogger.Info("Use case started", nil)

    result, err := uc.storage.GetActualizationStats(ctx)
    if err != nil {
        ucLogger.Error("Storage returned an error", err, nil)
        return nil, err
    }

    ucLogger.Info("Use case finished successfully", nil)
    
    return result, nil
}