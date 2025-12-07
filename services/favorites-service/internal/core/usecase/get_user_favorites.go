package usecase

import (
	"context"
	"favorites-service/internal/contextkeys"
	"favorites-service/internal/core/domain"
	"favorites-service/internal/core/port"
	"fmt"

	"github.com/google/uuid"
)

type GetUserFavoritesUseCase struct {
	favoritesRepo port.FavoritesRepositoryPort
	objectStorage port.ObjectStoragePort
}

func NewGetUserFavoritesUseCase(
	favoritesRepo port.FavoritesRepositoryPort,
	objectStorage port.ObjectStoragePort,
) *GetUserFavoritesUseCase {
	return &GetUserFavoritesUseCase{
		favoritesRepo: favoritesRepo,
		objectStorage: objectStorage,
	}
}

func (uc *GetUserFavoritesUseCase) Execute(ctx context.Context, userID uuid.UUID, limit, offset int) (*domain.PaginatedObjectsResult, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case": "GetUserFavorites",
		"user_id":  userID,
		"limit":    limit,
		"offset":   offset,
	})

	ucLogger.Info("Use case started", nil)
	
	// Шаг 1: Получаем пагинированный список ID из нашего собственного хранилища.
	paginatedIDs, err := uc.favoritesRepo.FindPaginatedByUser(ctx, userID, limit, offset)
	if err != nil {
		ucLogger.Error("Failed to get favorite IDs from repository", err, nil)
		return nil, fmt.Errorf("failed to get favorite IDs: %w", err)
	}

	if len(paginatedIDs.MasterObjectIDs) == 0 {
		// У пользователя нет избранных
		ucLogger.Info("No favorites on page", port.Fields{
			"current_page": offset/limit + 1,
			"total_count": paginatedIDs.TotalCount,
		})	
		return &domain.PaginatedObjectsResult{
			Objects:      []domain.ObjectCard{},
			TotalCount:   paginatedIDs.TotalCount, // <-- Используем реальное общее количество
			CurrentPage:  offset/limit + 1,
			ItemsPerPage: limit,
		}, nil
	}

	ucLogger.Info("User favorites found", port.Fields{
		"total_favorites": paginatedIDs.TotalCount,
		"ids_on_page":     len(paginatedIDs.MasterObjectIDs),
	})

	// Шаг 2: Идем в storage-service, чтобы "обогатить" эти ID данными.
	objects, err := uc.objectStorage.GetBestObjectsByMasterIDs(ctx, paginatedIDs.MasterObjectIDs)
	if err != nil {
		ucLogger.Error("Failed to get object details from storage service", err, nil)
		return nil, fmt.Errorf("failed to get object details from storage: %w", err)
	}
    
    // Шаг 3 (Важный): Сохраняем порядок.
    // storage-service не гарантирует порядок, а мы хотим показать избранное
    // в том порядке, в котором пользователь его добавлял (или в обратном).
    // Мы знаем правильный порядок из `paginatedIDs.MasterObjectIDs`.
    
    objectMap := make(map[uuid.UUID]domain.ObjectCard, len(objects))
    for _, obj := range objects {
        objectMap[obj.MasterObjectID] = obj
    }
    
    sortedObjects := make([]domain.ObjectCard, 0, len(paginatedIDs.MasterObjectIDs))
    for _, id := range paginatedIDs.MasterObjectIDs {
        if obj, ok := objectMap[id]; ok {
            sortedObjects = append(sortedObjects, obj)
        }
    }

	// Шаг 4: Формируем финальный результат.
	result := &domain.PaginatedObjectsResult{
		Objects:      sortedObjects,
		TotalCount:   paginatedIDs.TotalCount,
		CurrentPage:  offset/limit + 1,
		ItemsPerPage: limit,
	}

	ucLogger.Info("Use case finished successfully", nil)
	return result, nil
}