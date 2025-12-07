package usecases_port

import (
	"context"

	"github.com/google/uuid"
)

type ActualizeActiveObjectsUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, category string, limit int) (uuid.UUID, error)
}
