package rabbitmq

import (
	"time"

	"github.com/google/uuid"
)

type TaskInfo struct {
	Region   string `json:"region"`
	Category string `json:"category"`
    
    TaskID uuid.UUID `json:"task_id"`
}

type LinkTaskDTO struct {
	Source string    `json:"source"`
	AdID   int64     `json:"ad_id"`
	URL    string    `json:"ad_url"`
	TaskID uuid.UUID `json:"task_id"`
}


// ProcessedRealEstateEventDTO - это структура контракта
// Она точно соответствует JSON-схеме
type ProcessedRealEstateEventDTO struct {
    General     GeneralPropertyDTO `json:"general"`
    DetailsType string             `json:"details_type"`
    Details     interface{}        `json:"details"`

	TaskID 		uuid.UUID 			`json:"task_id"`
}

// GeneralPropertyDTO - часть контракта для общей информации
type GeneralPropertyDTO struct {
	Source     string `json:"source"`
	SourceAdID int64  `json:"sourceAdId"`
	// Category         string    `json:"category"`
	AdLink           string    `json:"adLink"`
	SaleType 		string    `json:"saleType"`
	Currency         string    `json:"currency"`
	Images           []string  `json:"images"`
	ListTime         time.Time `json:"listTime"`
	Description             string    `json:"description"`
	Title          string    `json:"title"`
	DealType         string    `json:"dealType"`
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	CityOrDistrict   string    `json:"cityOrDistrict"`
	Region           string    `json:"region"`
	PriceBYN         float64   `json:"priceBYN"`
	PriceUSD         float64   `json:"priceUSD"`
	PriceEUR         *float64  `json:"priceEUR,omitempty"`
	Address        string  `json:"address"`


	IsAgency        bool      `json:"isAgency"`
	SellerName     string  	  `json:"sellerName"`
	SellerDetails	map[string]interface{} `json:"sellerDetails"`

	Status		   string 		`json:"status"`
}


type ApartmentDetailsDTO struct {
    RoomsAmount         *int8    `json:"roomsAmount,omitempty"`
    FloorNumber         *int8    `json:"floorNumber,omitempty"`
    BuildingFloors      *int8    `json:"buildingFloors,omitempty"`
    TotalArea           *float64 `json:"totalArea,omitempty"`
    LivingSpaceArea     *float64 `json:"livingSpaceArea,omitempty"`
    KitchenArea         *float64 `json:"kitchenArea,omitempty"`
    YearBuilt           *int16   `json:"yearBuilt,omitempty"`
    WallMaterial        *string  `json:"wallMaterial,omitempty"`
    RepairState         *string  `json:"repairState,omitempty"`
    BathroomType        *string  `json:"bathroomType,omitempty"`
    Balcony             *string  `json:"balconyType,omitempty"`
    PricePerSquareMeter *float64 `json:"pricePerSquareMeter,omitempty"`
    Parameters          map[string]interface{} `json:"parameters"`
}


type HouseDetailsDTO struct {
    TotalArea         *float64 `json:"totalArea,omitempty"`
    PlotArea          *float64 `json:"plotArea,omitempty"`
    WallMaterial      *string  `json:"wallMaterial,omitempty"`
    YearBuilt         *int16   `json:"yearBuilt,omitempty"`
    LivingSpaceArea   *float64 `json:"livingSpaceArea,omitempty"`
    BuildingFloors    *int8    `json:"buildingFloors,omitempty"`
    RoomsAmount       *int8    `json:"roomsAmount,omitempty"`
    KitchenArea       *float64 `json:"kitchenArea,omitempty"`
    Electricity       *string  `json:"electricity,omitempty"`
    Water             *string  `json:"water,omitempty"`
    Heating           *string  `json:"heating,omitempty"`
    Sewage            *string  `json:"sewage,omitempty"`
    Gaz               *string  `json:"gaz,omitempty"`
    RoofMaterial      *string  `json:"roofMaterial,omitempty"`
    HouseType         *string  `json:"houseType,omitempty"`
    CompletionPercent *int8    `json:"completionPercent,omitempty"`
    Parameters        map[string]interface{} `json:"parameters"`
}


type CommercialDetailsDTO struct {
    IsNewCondition              *bool        `json:"isNewCondition,omitempty"` 
	PropertyType                *string      `json:"propertyType,omitempty"`       
	FloorNumber                 *int8        `json:"floorNumber,omitempty"`
	BuildingFloors              *int8        `json:"buildingFloors,omitempty"`           
	TotalArea                   *float64     `json:"totalArea,omitempty"`      
	CommercialImprovements      []string     `json:"commercialImprovements,omitempty"`             
	CommercialRepair            *string      `json:"commercialRepair,omitempty"`         	
	PricePerSquareMeter         *float64     `json:"pricePerSquareMeter,omitempty"`       
	RoomsRange                  []int8       `json:"roomsRange,omitempty"`           
	CommercialBuildingLocation  *string      `json:"commercialBuildingLocation,omitempty"`             
	CommercialRentType		    *string		 `json:"commercialRentType,omitempty"`    
	Parameters                  map[string]interface{}  `json:"parameters"`
}