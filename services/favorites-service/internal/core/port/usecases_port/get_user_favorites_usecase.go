package usecases_port

import (
	"context"
	"favorites-service/internal/core/domain"

	"github.com/google/uuid"
)

type GetUserFavoritesUseCasePort interface {
	// Возвращает срез ID объектов
	Execute(ctx context.Context, userID uuid.UUID, limit, offset int) (*domain.PaginatedObjectsResult, error)
}