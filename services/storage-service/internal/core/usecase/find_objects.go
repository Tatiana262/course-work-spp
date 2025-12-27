package usecase

import (
	"context"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"
)



type FindObjectsUseCase struct {
    storage port.PropertyStoragePort
}

func NewFindObjectsUseCase(storage port.PropertyStoragePort) *FindObjectsUseCase {
    return &FindObjectsUseCase{storage: storage}
}

func (uc *FindObjectsUseCase) Execute(ctx context.Context, filters domain.FindObjectsFilters, limit, offset int) (*domain.PaginatedResult, error) {
    // Получаем и обогащаем логгер
    logger := contextkeys.LoggerFromContext(ctx)
    ucLogger := logger.WithFields(port.Fields{
        "use_case": "FindObjects",
        "filters":  filters,
        "limit":    limit,
        "offset":   offset,
    })
    
    ucLogger.Info("Use case started", nil)

    // Выполняем основное действие
    result, err := uc.storage.FindWithFilters(ctx, filters, limit, offset)
    if err != nil {
        ucLogger.Error("Storage returned an error", err, nil)
        return nil, err // Просто пробрасываем ошибку дальше
    }

    // Логируем успешный результат
    ucLogger.Info("Use case finished successfully", port.Fields{
        "total_found": result.TotalCount,
        "items_on_page": len(result.Objects),
    })
    
    return result, nil
}