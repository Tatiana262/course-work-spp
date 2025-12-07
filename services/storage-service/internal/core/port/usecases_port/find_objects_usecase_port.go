package usecases_port

import (
	"context"
	"storage-service/internal/core/domain"
)

type FindObjectsUseCase interface {
	Execute(ctx context.Context, filters domain.FindObjectsFilters, limit, offset int) (*domain.PaginatedResult, error)
}