package port

import (
	"storage-service/internal/core/domain"
	"context"
)

type FilterOptionsRepositoryPort interface {
    GetPriceRange(ctx context.Context, req domain.FilterOptions) (*domain.RangeResult, error)
    GetDistinctRooms(ctx context.Context, req domain.FilterOptions) ([]int, error)
    GetDistinctWallMaterials(ctx context.Context, req domain.FilterOptions) ([]string, error)
    GetYearBuiltRange(ctx context.Context, req domain.FilterOptions) (*domain.RangeResult, error)
    // ... другие методы ...

	GetUniqueCategories(ctx context.Context) ([]domain.DictionaryItem, error)
    GetUniqueRegions(ctx context.Context) ([]domain.DictionaryItem, error)
    GetUniqueDealTypes(ctx context.Context) ([]domain.DictionaryItem, error)
}