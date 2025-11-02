package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)


// RealEstateRecord - это главная, агрегирующая структура для любого объекта недвижимости
type RealEstateRecord struct {
    General GeneralProperty
    Details interface{} 
}

// структура для хранения, отражение таблицы general_properties
type GeneralProperty struct {
	ID           uuid.UUID 			`db:"id"`
	Source       string    			`db:"source"`
	SourceAdID   int64     			`db:"source_ad_id"`
	CreatedAt    time.Time 			`db:"created_at"`
	UpdatedAt    time.Time 			`db:"updated_at"`

	Category         string     	`db:"category"`
	AdLink           string     	`db:"ad_link"`
	SaleType 		 string    		`db:"sale_type"`
	Currency         string     	`db:"currency"`
	Images           []string   	`db:"images"`
	ListTime         time.Time  	`db:"list_time"`
	Description     string     		`db:"description"`
	Title          	string     		`db:"title"`
	DealType         string     	`db:"deal_type"`
	Coordinates      string	 		`db:"coordinates"` 
	CityOrDistrict   string    		`db:"city_or_district"`
	Region           string    		`db:"region"`
	PriceBYN         float64   		`db:"price_byn"`
	PriceUSD         float64   		`db:"price_usd"`
	PriceEUR         *float64   	`db:"price_eur"`
	Address        	string 			`db:"address"`

	IsAgency        bool       		`db:"is_agency"`
	SellerName     string 			`db:"seller_name"`
	
	SellerDetails   json.RawMessage `db:"parameters"`

	Latitude    float64 `db:"-"`
	Longitude   float64 `db:"-"`
}

//структура для таблицы apartments
type Apartment struct {
	PropertyID            uuid.UUID       `db:"property_id"`
	RoomsAmount           *int16          `db:"rooms_amount"`
	FloorNumber           *int16          `db:"floor_number"`
	BuildingFloors        *int16          `db:"building_floors"`
	TotalArea             *float64        `db:"total_area"`
	LivingSpaceArea       *float64        `db:"living_space_area"`
	KitchenArea           *float64        `db:"kitchen_area"`
	YearBuilt             *int16          `db:"year_built"`
	WallMaterial          *string         `db:"wall_material"`
	RepairState           *string         `db:"repair_state"`
	BathroomType          *string         `db:"bathroom_type"`
	BalconyType           *string         `db:"balcony_type"`
	PricePerSquareMeter   *float64        `db:"price_per_square_meter"`
	
	Parameters            json.RawMessage `db:"parameters"`
}

// TODO
type House struct {
	PropertyID            uuid.UUID       			`db:"property_id"`
	TotalArea             *float64					`db:"total_area"`
	PlotArea              *float64					`db:"plot_area"` 
	WallMaterial          *string 					`db:"wall_material"`       
	Condition             *string    				`db:"condition"` 
	YearBuilt             *int16       				`db:"year_built"`
	LivingSpaceArea       *float64     				`db:"living_space_area"`
	BuildingFloors        *int16      				`db:"building_floors"`  
	RoomsAmount           *int16					`db:"rooms_amount"`
	KitchenSize           *float64    				`db:"kitchen_size"`	    
	Electricity           *bool						`db:"electricity"`         
	InGardeningCommunity  *bool  					`db:"in_gardening_community"`		         
	Water                 *string   				`db:"water"`     
	Heating               *string   				`db:"heating"`	      
	Sewage                *string    				`db:"sewage"` 	    
	Gaz                   *string   				`db:"gaz"`    
	RoofMaterial          *string   				`db:"roof_material"` 	     
	ContractNumberAndDate *string  					`db:"contract_number_and_date"`  	     
	HouseType             *string  					`db:"house_type"`		      
	Parameters            json.RawMessage 	`db:"parameters"`
}

type GarageAndParking struct {
	PropertyID          uuid.UUID				`db:"property_id"`
	PropertyType        *string					`db:"property_type"`  	 			    
	ParkingPlacesAmount *int16 		        	`db:"parking_places_amount"`
	TotalArea           *float64 	       		`db:"total_area"`
	Improvements        []string	      		`db:"improvements"`  
	Heating             *string 	      		`db:"heating"`  
	ParkingType         *string 	     		`db:"parking_type"`   
	Parameters          json.RawMessage 	`db:"parameters"`
}

type Room struct {
	PropertyID          	  uuid.UUID				`db:"property_id"`
	Condition                 *string				`db:"condition"`
	Bathroom                  *string				`db:"bathroom"`
	SuggestedRoomsAmount      *int16          		`db:"suggested_rooms_amount"`
	RoomsAmount               *int16            	`db:"rooms_amount"`
	FloorNumber               *int16            	`db:"floor_number"`
	BuildingFloors            *int16          		`db:"building_floors"`
	TotalArea                 *float64        		`db:"total_area"`
	IsBalcony                 *bool           		`db:"is_balcony"`
	RentalType                *string         		`db:"rental_type"`
	LivingSpaceArea           *float64        		`db:"living_space_area"`
	FlatRepair                *string         		`db:"flat_repair"`
	IsFurniture               *bool           		`db:"is_furniture"`
	KitchenSize               *float64        		`db:"kitchen_size"`
	KitchenItems              []string        		`db:"kitchen_items"`
	BathItems                 []string        		`db:"bath_items"`
	FlatRentForWhom           []string        		`db:"flat_rent_for_whom"`
	FlatWindowsSide           []string        		`db:"flat_windows_side"`
	YearBuilt                 *int16          		`db:"year_built"`
	WallMaterial              *string         		`db:"wall_material"`
	FlatImprovement           []string        		`db:"flat_improvement"`
	RoomType                  *string         		`db:"room_type"`
	ContractNumberAndDate     *string         		`db:"contract_number_and_date"`
	FlatBuildingImprovements  []string        		`db:"flat_building_improvements"`
	Parameters                json.RawMessage `db:"parameters"`
}

type Commercial struct {
	PropertyID          	  uuid.UUID					`db:"property_id"`
	Condition                  *string         			`db:"condition"`
	PropertyType               *string         			`db:"property_type"`
	FloorNumber                *int16          			`db:"floor_number"`
	BuildingFloors             *int16          			`db:"building_floors"`
	TotalArea                  *float64        			`db:"total_area"`
	CommercialImprovements     []string        			`db:"commercial_improvements"`
	CommercialRepair           *string         			`db:"commercial_repair"`
	IsPartlySellOrRent         *bool         			`db:"partly_sell"`
	PricePerSquareMeter        *float64        			`db:"price_per_square_meter"`
	ContractNumberAndDate      *string         			`db:"contract_number_and_date"`
	RoomsAmount                *int16          			`db:"rooms_amount"`
	CommercialBuildingLocation *string         			`db:"commercial_building_location"`
	CommercialRentType		   *string		   			`db:"commercial_rent_type"`
	Parameters                 json.RawMessage	`db:"parameters"`		 
}

type Plot struct {
	PropertyID            uuid.UUID					`db:"property_id"`
	PlotArea              *float64  				`db:"plot_area"`      
	InGardeningCommunity  *bool     				`db:"in_gardening_community"`      
	PropertyRights        *string   				`db:"property_rights"`      
	Electricity           *string 					`db:"electricity"`        
	Water                 *string 					`db:"water"`        
	Gaz                   *string 					`db:"gaz"`        
	Sewage                *string  					`db:"sewage"`       
	IsOutbuildings        *bool    					`db:"is_outbuildings"`       
	OutbuildingsType      []string 					`db:"outbuildings_type"`       
	ContractNumberAndDate *string      				`db:"contract_number_and_date"`   
	Parameters            json.RawMessage	`db:"parameters"`	
}

type NewBuilding struct {
	PropertyID          uuid.UUID				`db:"property_id"`
	Deadline            *string         		`db:"deadline"`
	RoomOptions         []int16         		`db:"room_options"`
	Builder             *string         		`db:"builder"`
	ShareParticipation  *bool           		`db:"share_participation"`
	FloorOptions        []int16         		`db:"floor_options"`
	WallMaterial        *string         		`db:"wall_material"`
	CeilingHeight       *string         		`db:"flat_ceiling_height"`
	LayoutOptions       []string        		`db:"layout_options"`
	WithFinishing       *bool           		`db:"with_finishing"`
	Parameters          json.RawMessage	`db:"parameters"`
}