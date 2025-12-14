package usecases_port

import (
	"context"
	"storage-service/internal/core/domain"
)

type GetFilterOptionsUseCase interface {
	Execute(ctx context.Context, req domain.FindObjectsFilters) (*domain.FilterOptionsResult, error)
}