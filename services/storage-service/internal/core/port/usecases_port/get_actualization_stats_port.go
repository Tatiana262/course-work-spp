package usecases_port

import (
	"context"
	"storage-service/internal/core/domain"
)

type GetActualizationStats interface {
	Execute(ctx context.Context) ([]domain.StatsByCategory, error)
}