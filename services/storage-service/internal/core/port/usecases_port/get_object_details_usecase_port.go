package usecases_port

import (
	"context"
	"storage-service/internal/core/domain"
	"github.com/google/uuid"
)

type GetObjectDetailsUseCase interface {
	Execute(ctx context.Context, objectID uuid.UUID) (*domain.PropertyDetailsView, error)
}