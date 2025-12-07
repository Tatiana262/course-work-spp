package usecases_port

import (
	"context"

	"github.com/google/uuid"
)

type FindNewObjectsUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, categories []string, regions []string)  (uuid.UUID, error)
}