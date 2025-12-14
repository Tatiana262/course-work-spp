package port

import (
	"context"
	"favorites-service/internal/core/domain"

	"github.com/google/uuid"
)

// FavoritesRepositoryPort - контракт для адаптера, работающего с локальной БД избранного.
type FavoritesRepositoryPort interface {
	Add(ctx context.Context, userID, masterObjectID uuid.UUID) error
	Remove(ctx context.Context, userID, masterObjectID uuid.UUID) error
	FindPaginatedByUser(ctx context.Context, userID uuid.UUID, limit, offset int) (*domain.PaginatedFavoriteIDs, error)
	FindFavoritesIdsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}