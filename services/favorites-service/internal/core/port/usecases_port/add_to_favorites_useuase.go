package usecases_port

import (
	"context"

	"github.com/google/uuid"
)

type AddToFavoritesUseCasePort interface {
	Execute(ctx context.Context, userID, objectID uuid.UUID) error
}