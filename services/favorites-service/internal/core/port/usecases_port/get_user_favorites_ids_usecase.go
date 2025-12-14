package usecases_port

import (
	"context"

	"github.com/google/uuid"
)

type GetUserFavoritesIdsUseCasePort interface {
	// Возвращает срез ID объектов
	Execute(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}