package usecase

import (
	"context"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"
)



type GetFilterOptionsUseCase struct {
    storage port.FilterOptionsRepositoryPort // Новый порт
}

func NewGetFilterOptionsUseCase(storage port.FilterOptionsRepositoryPort) *GetFilterOptionsUseCase {
    return &GetFilterOptionsUseCase{storage: storage}
}

// Execute - основной метод, который собирает опции в зависимости от категории.
func (uc *GetFilterOptionsUseCase) Execute(ctx context.Context, req domain.FilterOptions) (map[string]domain.FilterOption, error) {
    logger := contextkeys.LoggerFromContext(ctx)
    ucLogger := logger.WithFields(port.Fields{
        "use_case": "GetFilterOptionsUseCase",
    })

    ucLogger.Info("Use case started", nil)
    
    // Создаем map для хранения финального результата.
    result := make(map[string]domain.FilterOption)
    var err error

    // --- Общие фильтры (цена) ---
    priceRange, err := uc.storage.GetPriceRange(ctx, req)
    if err == nil {
        result["price_byn"] = domain.FilterOption{Type: "range", Min: priceRange.Min, Max: priceRange.Max}
    }
    
    // --- Специфичные для категории фильтры ---
    switch req.Category {
    case "apartment":
        // Количество комнат
        rooms, err := uc.storage.GetDistinctRooms(ctx, req)
        if err == nil {
            result["rooms"] = domain.FilterOption{Type: "checkbox", Options: toInterfaceSlice(rooms)}
        }
        // Материал стен
        materials, err := uc.storage.GetDistinctWallMaterials(ctx, req)
        if err == nil {
            result["wall_material"] = domain.FilterOption{Type: "select", Options: toInterfaceSlice(materials)}
        }
        // Год постройки
        yearRange, err := uc.storage.GetYearBuiltRange(ctx, req)
        if err == nil {
            result["year_built"] = domain.FilterOption{Type: "range", Min: yearRange.Min, Max: yearRange.Max}
        }
    
    case "house":
         // Логика для домов: материал стен, этажность, площадь участка...
         // ...

    // ... другие категории ...
    }
    
    // ВАЖНО: Мы не возвращаем ошибку, если не удалось получить один из фильтров.
    // Лучше показать 9 из 10 фильтров, чем ни одного. Ошибки нужно логировать.
    return result, nil
}

// Вспомогательная функция для преобразования срезов.
func toInterfaceSlice[T any](slice []T) []interface{} {
    result := make([]interface{}, len(slice))
    for i, v := range slice {
        result[i] = v
    }
    return result
}