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

func NewGetObjectByIDUseCase(storage port.PropertyStoragePort) *GetObjectByIDUseCase {
    return &GetObjectByIDUseCase{storage: storage}
}

// FindIDsForActualization - пример метода, который вернет ID для актуализации.
// Вы можете добавить сюда параметры: лимит, оффсет, фильтры и т.д.
//TODO: попробовать заменить на uuid
func (uc *GetObjectByIDUseCase) FindObjectByIDForActualization(ctx context.Context, id string) (*domain.PropertyBasicInfo, error) {
	logger := contextkeys.LoggerFromContext(ctx)
    ucLogger := logger.WithFields(port.Fields{
        "use_case": "FindObjectByIDForActualization",
        "id": id,
    })

    ucLogger.Info("Use case started", nil)

	result, err := uc.storage.GetObjectByIDForActualization(ctx, id)
    if err != nil {
        ucLogger.Error("Storage returned an error", err, nil)
        return nil, err
    }

    ucLogger.Info("Use case finished successfully", nil)
    
    return result, nil

}