package usecase

import (
	"context"
	"favorites-service/internal/contextkeys"
	"favorites-service/internal/core/port"

	"github.com/google/uuid"
)

type AddToFavoritesUseCase struct {
	repo port.FavoritesRepositoryPort
}

func NewAddToFavoritesUseCase(repo port.FavoritesRepositoryPort) *AddToFavoritesUseCase {
	return &AddToFavoritesUseCase{repo: repo}
}

func (uc *AddToFavoritesUseCase) Execute(ctx context.Context, userID, objectID uuid.UUID) error {
	// Здесь может быть бизнес-логика. Например, проверка, существует ли
	// такой objectID (сходив в real-estate-objects-service), или проверка
	// на максимальное количество объектов в избранном.
	//
	// Пока что логика проста: просто вызываем репозиторий.
	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case":         "AddToFavorites",
		"user_id":          userID,
		"master_object_id": objectID,
	})
	
	ucLogger.Info("Use case started", nil)

	err := uc.repo.Add(ctx, userID, objectID)
	if err != nil {
		ucLogger.Error("Repository returned an error", err, nil)
		return err // Просто пробрасываем ошибку, она уже залогирована в репозитории
	}

	ucLogger.Info("Use case finished successfully", nil)
	return nil
}