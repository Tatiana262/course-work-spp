package usecases_port

import (
	"context"

	"github.com/google/uuid"
)

type ActualizeObjectByIdUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, id string) (uuid.UUID, error)
}
