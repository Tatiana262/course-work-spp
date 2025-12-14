package usecase

import (
	"context"
	// "log"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"
	"strings"
)

type GetDictionariesUseCase struct {
	repo port.FilterOptionsRepositoryPort
}

func NewGetDictionariesUseCase(repo port.FilterOptionsRepositoryPort) *GetDictionariesUseCase {
	return &GetDictionariesUseCase{repo: repo}
}

// Execute получает список имен справочников и возвращает их содержимое.
func (uc *GetDictionariesUseCase) Execute(ctx context.Context, names []string) (map[string][]domain.DictionaryItem, error) {

	logger := contextkeys.LoggerFromContext(ctx)
    ucLogger := logger.WithFields(port.Fields{
        "use_case": "GetDictionariesUseCase",
    })

    ucLogger.Info("Use case started", nil)
	
	// Создаем map для хранения финального результата.
	result := make(map[string][]domain.DictionaryItem)
	
	// Используем map для удобства проверки, какие справочники запрошены.
	namesMap := make(map[string]bool)
	for _, name := range names {
		namesMap[strings.TrimSpace(name)] = true
	}
	
	// Если запрошены категории или все справочники
	if namesMap["categories"] || len(namesMap) == 0 {
		categories, err := uc.repo.GetUniqueCategories(ctx)
		if err != nil {
			ucLogger.Error("Storage returned an error while getting unique categories", err, nil)
		} else {
			result["categories"] = categories
		}
	}

	// Если запрошены регионы или все справочники
	if namesMap["regions"] || len(namesMap) == 0 {
		regions, err := uc.repo.GetUniqueRegions(ctx)
		if err != nil {
			ucLogger.Error("Storage returned an error while getting unique regions", err, nil)
		} else {
			result["regions"] = regions
		}
	}

	// Если запрошены типы сделок или все справочники
	if namesMap["deal_types"] || len(namesMap) == 0 {
		dealTypes, err := uc.repo.GetUniqueDealTypes(ctx)
		// log.Println("!!!!!", dealTypes, "!!!!!!!")
		if err != nil {
			ucLogger.Error("Storage returned an error while getting unique deal types", err, nil)
		} else {
			result["deal_types"] = dealTypes
		}
	}

	return result, nil
}