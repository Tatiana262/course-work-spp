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
func (uc *GetFilterOptionsUseCase) Execute(ctx context.Context, req domain.FindObjectsFilters) (*domain.FilterOptionsResult, error) {
    logger := contextkeys.LoggerFromContext(ctx)
    ucLogger := logger.WithFields(port.Fields{
        "use_case": "GetFilterOptionsUseCase",
    })

    ucLogger.Info("Use case started", nil)
    
    // Создаем map для хранения финального результата.
    resultOptions := make(map[string]domain.FilterOption)
    
    // --- Фаза 1: Получаем "умный" диапазон цен, который зависит от всех фильтров ---
	priceRange, err := uc.storage.GetPriceRange(ctx, req)
	if err == nil {
		resultOptions["price"] = domain.FilterOption{ Min: priceRange.Min, Max: priceRange.Max}
	} else {
		ucLogger.Error("WARN: Failed to get price range", err, nil)
	}

    count, err := uc.storage.GetTotalCount(ctx, req)
	if err != nil {
		ucLogger.Error("Failed to get total count", err, nil)
		// Не критично, можем вернуть 0, но лучше знать об ошибке
	}

    if req.Category == "" {
		return &domain.FilterOptionsResult{
            Options: resultOptions,
            Count: count,
        }, nil
	}

    if req.Region != "" {
        cities, err := uc.storage.GetUniqueCitiesByRegion(ctx, req.Region)
        if err == nil && len(cities) > 0 {
			resultOptions["cities"] = domain.FilterOption{ Options: toInterfaceSlice(cities)}
		}
    }

    // --- Специфичные для категории фильтры ---
    switch req.Category {
    case "apartment":
        // Количество комнат
        rooms, err := uc.storage.GetApartmentDistinctRooms(ctx)
		if err == nil && len(rooms) > 0 {
			resultOptions["rooms"] = domain.FilterOption{Options: rooms}
		}
        // Этаж
        floor, err := uc.storage.GetApartmentFloorsRange(ctx)
        if err == nil {
            resultOptions["floor"] = domain.FilterOption{ Min: floor.Min, Max: floor.Max}
        }
        // Этажность здания
        buildingFloor, err := uc.storage.GetApartmentBuildingFloorsRange(ctx)
        if err == nil {
            resultOptions["building_floor"] = domain.FilterOption{ Min: buildingFloor.Min, Max: buildingFloor.Max}
        }
        // Общая площадь
        totalArea, err := uc.storage.GetApartmentTotalAreaRange(ctx)
        if err == nil {
            resultOptions["total_area"] = domain.FilterOption{Min: totalArea.Min, Max: totalArea.Max}
        }
        // Жилая площадь
        livingSpaceArea, err := uc.storage.GetApartmentLivingSpaceAreaRange(ctx)
        if err == nil {
            resultOptions["living_space_area"] = domain.FilterOption{ Min: livingSpaceArea.Min, Max: livingSpaceArea.Max}
        }
        // Площадь кухни
        kitchenArea, err := uc.storage.GetApartmentKitchenAreaRange(ctx)
        if err == nil {
            resultOptions["kitchen_area"] = domain.FilterOption{ Min: kitchenArea.Min, Max: kitchenArea.Max}
        }
        // Год постройки
        yearRange, err := uc.storage.GetApartmentYearBuiltRange(ctx)
        if err == nil {
            resultOptions["year_built"] = domain.FilterOption{ Min: yearRange.Min, Max: yearRange.Max}
        }
        // Материал стен
        materials, err := uc.storage.GetApartmentDistinctWallMaterials(ctx)
        if err == nil {
            resultOptions["wall_materials"] = domain.FilterOption{ Options: materials}
        }
        // Состояние ремонта
        repairStates, err := uc.storage.GetApartmentDistinctRepairStates(ctx)
        if err == nil {
            resultOptions["repair_states"] = domain.FilterOption{ Options: repairStates}
        }
        // Тип санузла
        bathroomTypes, err := uc.storage.GetApartmentDistinctBathroomTypes(ctx)
        if err == nil {
            resultOptions["bathroom_types"] = domain.FilterOption{ Options: bathroomTypes}
        }
        // Тип балкона
        balconyTypes, err := uc.storage.GetApartmentDistinctBalconyTypes(ctx)
        if err == nil {
            resultOptions["balcony_types"] = domain.FilterOption{ Options: balconyTypes}
        }
    
    case "house":
        // Количество комнат
        rooms, err := uc.storage.GetHouseDistinctRooms(ctx)
        if err == nil && len(rooms) > 0 {
            resultOptions["rooms"] = domain.FilterOption{Options: rooms}
        }
        // Тип
        houseTypes, err := uc.storage.GetHouseDistinctTypes(ctx)
        if err == nil {
            resultOptions["house_types"] = domain.FilterOption{Options: houseTypes}
        }
        // Общая площадь
        totalArea, err := uc.storage.GetHouseTotalAreaRange(ctx)
        if err == nil {
            resultOptions["total_area"] = domain.FilterOption{Min: totalArea.Min, Max: totalArea.Max}
        }
        // Жилая площадь
        livingSpaceArea, err := uc.storage.GetHouseLivingSpaceAreaRange(ctx)
        if err == nil {
            resultOptions["living_space_area"] = domain.FilterOption{Min: livingSpaceArea.Min, Max: livingSpaceArea.Max}
        }
        // Площадь кухни
        kitchenArea, err := uc.storage.GetHouseKitchenAreaRange(ctx)
        if err == nil {
            resultOptions["kitchen_area"] = domain.FilterOption{Min: kitchenArea.Min, Max: kitchenArea.Max}
        }
        // Площадь участка
        plotArea, err := uc.storage.GetHousePlotAreaRange(ctx)
        if err == nil {
            resultOptions["plot_area"] = domain.FilterOption{Min: plotArea.Min, Max: plotArea.Max}
        }
        // Площадь участка
        floor, err := uc.storage.GetHouseFloorsRange(ctx)
        if err == nil {
            resultOptions["floor"] = domain.FilterOption{Min: floor.Min, Max: floor.Max}
        }
        // Год постройки
        yearRange, err := uc.storage.GetHouseYearBuiltRange(ctx)
        if err == nil {
            resultOptions["year_built"] = domain.FilterOption{Min: yearRange.Min, Max: yearRange.Max}
        }
        // Материал стен
        wallMaterials, err := uc.storage.GetHouseDistinctWallMaterials(ctx)
        if err == nil {
            resultOptions["wall_materials"] = domain.FilterOption{Options: wallMaterials}
        }
        // Материал крыши
        roofMaterials, err := uc.storage.GetHouseDistinctRoofMaterials(ctx)
        if err == nil {
            resultOptions["roof_materials"] = domain.FilterOption{Options: roofMaterials}
        }
        // Вода
        waterTypes, err := uc.storage.GetHouseDistinctWaterTypes(ctx)
        if err == nil {
            resultOptions["water_types"] = domain.FilterOption{Options: waterTypes}
        }

        // Отопление
        heatingTypes, err := uc.storage.GetHouseDistinctHeatingTypes(ctx)
        if err == nil {
            resultOptions["heating_types"] = domain.FilterOption{Options: heatingTypes}
        }
        // Электричество
        electricityTypes, err := uc.storage.GetHouseDistinctElectricityTypes(ctx)
        if err == nil {
            resultOptions["electricity_types"] = domain.FilterOption{Options: electricityTypes}
        }
        // Канализация
        sewageTypes, err := uc.storage.GetHouseDistinctSewageTypes(ctx)
        if err == nil {
            resultOptions["sewage_types"] = domain.FilterOption{Options: sewageTypes}
        }
        // Газ
        gazTypes, err := uc.storage.GetHouseDistinctGazTypes(ctx)
        if err == nil {
            resultOptions["gaz_types"] = domain.FilterOption{Options: gazTypes}
        }
    case "commercial":
        commercialTypes, err := uc.storage.GetCommercialDistinctTypes(ctx)
        if err == nil {
            resultOptions["commercial_types"] = domain.FilterOption{Options: commercialTypes}
        }
        // Этаж
        floor, err := uc.storage.GetCommercialFloorsRange(ctx)
        if err == nil {
            resultOptions["floor"] = domain.FilterOption{ Min: floor.Min, Max: floor.Max}
        }
        // Этажность здания
        buildingFloor, err := uc.storage.GetCommercialBuildingFloorsRange(ctx)
        if err == nil {
            resultOptions["building_floor"] = domain.FilterOption{ Min: buildingFloor.Min, Max: buildingFloor.Max}
        }
        // Общая площадь
        totalArea, err := uc.storage.GetCommercialTotalAreaRange(ctx)
        if err == nil {
            resultOptions["total_area"] = domain.FilterOption{Min: totalArea.Min, Max: totalArea.Max}
        }
        commercialImprovements, err := uc.storage.GetCommercialImprovements(ctx)
        if err == nil {
            resultOptions["commercial_improvements"] = domain.FilterOption{Options: commercialImprovements}
        }
        // Ремонт
        commercialRepair, err := uc.storage.GetCommercialRepairs(ctx)
        if err == nil {
            resultOptions["commercial_repairs"] = domain.FilterOption{ Options: commercialRepair}
        }
        // Местоположение
        commercialLocation, err := uc.storage.GetCommercialLocations(ctx)
        if err == nil {
            resultOptions["commercial_locations"] = domain.FilterOption{Options: commercialLocation}
        }
        // Раздельные помещения
        commercialRooms, err := uc.storage.GetCommercialRoomsRange(ctx)
        if err == nil {
            resultOptions["commercial_rooms"] = domain.FilterOption{Min: commercialRooms.Min, Max: commercialRooms.Max}
        }
    }
    
    // не возвращаем ошибку, если не удалось получить один из фильтров
    return &domain.FilterOptionsResult{
        Options: resultOptions,
        Count: count,
    }, nil
}

// Вспомогательная функция для преобразования срезов.
func toInterfaceSlice[T any](slice []T) []interface{} {
    result := make([]interface{}, len(slice))
    for i, v := range slice {
        result[i] = v
    }
    return result
}