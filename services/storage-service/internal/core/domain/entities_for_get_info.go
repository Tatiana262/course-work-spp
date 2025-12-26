package domain

import (
	"time"

	"github.com/google/uuid"
)

// FindObjectsFilters - структура для передачи всех возможных фильтров.
type FindObjectsFilters struct {
	Category string
    DealType string
    PriceCurrency string
    PriceMin *float64
	PriceMax *float64
    Region          string    // "Брестская область"
    CityOrDistrict  string    // "Брест", "Пинск"
    Street          string    // "Васнецова", "Московская"

    // Дополнительно  
    Rooms          []int 

    TotalAreaMin   *float64
    TotalAreaMax   *float64

    LivingSpaceAreaMin *float64
    LivingSpaceAreaMax *float64

    KitchenAreaMin *float64
    KitchenAreaMax *float64

    YearBuiltMin   *int     
	YearBuiltMax   *int

    WallMaterials  []string 

    // Только для квартир
	FloorMin       *int     
	FloorMax       *int

    FloorBuildingMin       *int     
	FloorBuildingMax       *int    
   
    RepairState    []string
    BathroomType   []string
    BalconyType    []string

    // PricePerSquareMeterMin *float64 
	// PricePerSquareMeterMax *float64		

    // Только для домов
    HouseTypes   []string

    PlotAreaMin *float64
    PlotAreaMax *float64

    TotalFloors  []string
    RoofMaterials  []string 
    WaterConditions []string
    HeatingConditions []string
    ElectricityConditions []string
    SewageConditions []string
    GazConditions []string

    // коммерция
    PropertyType string
    CommercialImprovements []string
    CommercialRepairs []string
    CommercialLocation []string

    CommercialRoomsMin *int
    CommercialRoomsMax *int
}

// PaginatedResult - стандартная структура для ответа с пагинацией.
type PaginatedResult struct {
    Objects      []GeneralPropertyInfo // Возвращаем только общую информацию для списка
    TotalCount   int                      // Общее количество найденных объектов
    CurrentPage  int
    ItemsPerPage int
}


// TODO
type GeneralPropertyInfo struct {
    MasterObjectID	string	
	ID           uuid.UUID 			
	Source       string    			
	SourceAdID   int64     			
	UpdatedAt    time.Time 	
    CreatedAt  time.Time		
	Category         string
    AdLink     string    
    SaleType   string
    Currency   string
    Images           []string  
    ListTime  time.Time 
    Description string
    Title       string
	DealType         string 
    CityOrDistrict string  
    Region      string  	
	
	PriceBYN         float64   		
	PriceUSD         float64   		
	PriceEUR         *float64   	
	
	Address  string
    IsAgency bool
    SellerName string
    Status   string 

    SellerDetails interface{}
}


type DuplicatesInfo struct {
	ID           uuid.UUID 			
	Source       string    			 
	AdLink       string  
	IsSourceDuplicate bool
	DealType	 string
}

type PropertyDetailsView struct {
    MainProperty  GeneralPropertyInfo // Используем полную структуру
    Details       interface{}       // *Apartment, *House...
    RelatedOffers []DuplicatesInfo // Список всех других предложений
}