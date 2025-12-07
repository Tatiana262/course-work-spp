package usecase

import (
	"context"
	"favorites-service/internal/contextkeys"
	"favorites-service/internal/core/port"

	"github.com/google/uuid"
)

type RemoveFromFavoritesUseCase struct {
	repo port.FavoritesRepositoryPort
}

func NewRemoveFromFavoritesUseCase(repo port.FavoritesRepositoryPort) *RemoveFromFavoritesUseCase {
	return &RemoveFromFavoritesUseCase{repo: repo}
}

func (uc *RemoveFromFavoritesUseCase) Execute(ctx context.Context, userID, objectID uuid.UUID) error {
	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case":         "RemoveFromFavorites",
		"user_id":          userID,
		"master_object_id": objectID,
	})

	ucLogger.Info("Use case started", nil)

	err := uc.repo.Remove(ctx, userID, objectID)
	if err != nil {
		ucLogger.Error("Repository returned an error", err, nil)
		return err
	}

	ucLogger.Info("Use case finished successfully", nil)
	return nil
}