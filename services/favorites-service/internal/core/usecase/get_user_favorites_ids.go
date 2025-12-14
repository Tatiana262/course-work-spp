package usecase

import (
	"context"
	"favorites-service/internal/contextkeys"
	"favorites-service/internal/core/port"
	"fmt"

	"github.com/google/uuid"
)

type GetUserFavoritesIdsUseCase struct {
	favoritesRepo port.FavoritesRepositoryPort
}

func NewGetUserFavoritesIdsUseCase(
	favoritesRepo port.FavoritesRepositoryPort,
) *GetUserFavoritesIdsUseCase {
	return &GetUserFavoritesIdsUseCase{
		favoritesRepo: favoritesRepo,
	}
}

func (uc* GetUserFavoritesIdsUseCase) Execute(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case": "GetUserFavorites",
		"user_id":  userID,
	})

	ucLogger.Info("Use case started", nil)

	ids, err := uc.favoritesRepo.FindFavoritesIdsByUser(ctx, userID)
	if err != nil {
		ucLogger.Error("Failed to get favorite IDs from repository", err, nil)
		return nil, fmt.Errorf("failed to get favorite IDs: %w", err)
	}

	return ids, nil
}