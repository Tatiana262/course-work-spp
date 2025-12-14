package port

import (
	"storage-service/internal/core/domain"
	"context"
)

type FilterOptionsRepositoryPort interface {
    // GetTotalCount(ctx context.Context, req domain.FilterOptions) (int, error)
    // GetApartmentDistinctRooms(ctx context.Context, category string) ([]interface{}, error)
    // GetApartmentDistinctWallMaterials(ctx context.Context, category string) ([]interface{}, error)

    // GetPriceRange(ctx context.Context, req domain.FilterOptions) (*domain.RangeResult, error)
    // GetDistinctRooms(ctx context.Context, req domain.FilterOptions) ([]int, error)
    
    // GetYearBuiltRange(ctx context.Context, req domain.FilterOptions) (*domain.RangeResult, error)
    // // ... другие методы ...

	// GetUniqueCategories(ctx context.Context) ([]domain.DictionaryItem, error)
    // GetUniqueRegions(ctx context.Context) ([]domain.DictionaryItem, error)
    // GetUniqueDealTypes(ctx context.Context) ([]domain.DictionaryItem, error)

    // общие
    GetPriceRange(ctx context.Context, req domain.FindObjectsFilters) (*domain.RangeResult, error)
    GetUniqueCategories(ctx context.Context) ([]domain.DictionaryItem, error)
    GetUniqueRegions(ctx context.Context) ([]domain.DictionaryItem, error)
    GetUniqueDealTypes(ctx context.Context) ([]domain.DictionaryItem, error)
    GetUniqueCitiesByRegion(ctx context.Context, region string) ([]string, error)

    // общее количество
    GetTotalCount(ctx context.Context, req domain.FindObjectsFilters) (int, error)

    // квартиры
    GetApartmentDistinctRooms(ctx context.Context) ([]interface{}, error)
    GetApartmentFloorsRange(ctx context.Context) (*domain.RangeResult, error)
    GetApartmentBuildingFloorsRange(ctx context.Context) (*domain.RangeResult, error) 
    GetApartmentTotalAreaRange(ctx context.Context) (*domain.RangeResult, error)
    GetApartmentLivingSpaceAreaRange(ctx context.Context) (*domain.RangeResult, error)
    GetApartmentKitchenAreaRange(ctx context.Context) (*domain.RangeResult, error)
    GetApartmentYearBuiltRange(ctx context.Context) (*domain.RangeResult, error)
    GetApartmentDistinctWallMaterials(ctx context.Context) ([]interface{}, error)
    GetApartmentDistinctRepairStates(ctx context.Context) ([]interface{}, error)
    GetApartmentDistinctBathroomTypes(ctx context.Context) ([]interface{}, error)
    GetApartmentDistinctBalconyTypes(ctx context.Context) ([]interface{}, error)

    // дома
    GetHouseDistinctRooms(ctx context.Context) ([]interface{}, error)
    GetHouseDistinctTypes(ctx context.Context) ([]interface{}, error)
    GetHouseTotalAreaRange(ctx context.Context) (*domain.RangeResult, error)
    GetHouseLivingSpaceAreaRange(ctx context.Context) (*domain.RangeResult, error)
    GetHouseKitchenAreaRange(ctx context.Context) (*domain.RangeResult, error)
    GetHousePlotAreaRange(ctx context.Context) (*domain.RangeResult, error)
    GetHouseFloorsRange(ctx context.Context) (*domain.RangeResult, error)
    GetHouseYearBuiltRange(ctx context.Context) (*domain.RangeResult, error)  
    GetHouseDistinctWallMaterials(ctx context.Context) ([]interface{}, error)
    GetHouseDistinctRoofMaterials(ctx context.Context) ([]interface{}, error)
    GetHouseDistinctWaterTypes(ctx context.Context) ([]interface{}, error)
    GetHouseDistinctHeatingTypes(ctx context.Context) ([]interface{}, error)
    GetHouseDistinctElectricityTypes(ctx context.Context) ([]interface{}, error)
    GetHouseDistinctSewageTypes(ctx context.Context) ([]interface{}, error)
    GetHouseDistinctGazTypes(ctx context.Context) ([]interface{}, error)
}