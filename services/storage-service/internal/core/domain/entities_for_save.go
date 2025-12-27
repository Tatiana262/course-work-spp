package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)


// RealEstateRecord - это главная, агрегирующая структура для любого объекта недвижимости
type RealEstateRecord struct {
    General GeneralProperty
    Details interface{} // Сюда будет помещен указатель на Apartment, House
}

// dbGeneralProperty - это структура для хранения
type GeneralProperty struct {
	ID           uuid.UUID 			
	Source       string    			
	SourceAdID   int64     			
	CreatedAt    time.Time 			
	UpdatedAt    time.Time 			

	Category         string     	
	AdLink           string     	
	SaleType 		 string    		
	Currency         string     	
	Images           []string   	
	ListTime         time.Time  	
	Description     string     		
	Title          	string     		
	DealType         string     	
	Coordinates      string	 		
	CityOrDistrict   string    		
	Region           string    		
	PriceBYN         float64   		
	PriceUSD         float64   		
	PriceEUR         *float64   	
	Address        	string 			

	IsAgency        bool       		
	SellerName     string 			
	
	SellerDetails   json.RawMessage 

	Status 			string			

	Latitude    float64 
	Longitude   float64 
}

// dbApartment - структура для таблицы `apartments`
type Apartment struct {
	PropertyID            uuid.UUID       `json:"-"`
	RoomsAmount           *int8          `json:"rooms_amount"`
	FloorNumber           *int8          `json:"floor_number"`
	BuildingFloors        *int8          `json:"building_floors"`
	TotalArea             *float64        `json:"total_area"`
	LivingSpaceArea       *float64        `json:"living_space_area"`
	KitchenArea           *float64        `json:"kitchen_area"`
	YearBuilt             *int16          `json:"year_built"`
	WallMaterial          *string         `json:"wall_material"`
	RepairState           *string         `json:"repair_state"`
	BathroomType          *string         `json:"bathroom_type"`
	BalconyType           *string         `json:"balcony_type"`
	PricePerSquareMeter   *float64        `json:"price_per_square_meter"`
	
	IsNewCondition        *bool 		  `json:"is_new_condition"`
	Parameters            json.RawMessage `json:"parameters"`
}

type House struct {
	PropertyID            uuid.UUID       			`json:"-"`
	TotalArea             *float64					`json:"total_area"`
	PlotArea              *float64					`json:"plot_area"` 
	WallMaterial          *string 					`json:"wall_material"` 
	YearBuilt             *int16       				`json:"year_built"`  
	LivingSpaceArea       *float64     				`json:"living_space_area"`
	BuildingFloors        *int8      				`json:"building_floors"`  
	RoomsAmount           *int8						`json:"rooms_amount"`
	KitchenArea           *float64    				`json:"kitchen_area"`	  
	Electricity           *string					`json:"electricity"`  
	Water                 *string   				`json:"water"`
	Heating               *string   				`json:"heating"`	      
	Sewage                *string    				`json:"sewage"` 	    
	Gaz                   *string   				`json:"gaz"`    
	RoofMaterial          *string   				`json:"roof_material"` 	
	HouseType             *string  					`json:"house_type"`
	
	CompletionPercent 	  *int8						`json:"completion_percent"`
	IsNewCondition        *bool 					`json:"is_new_condition"`

	// Condition             *string    				`db:"condition"`     
	// InGardeningCommunity  *bool  					`db:"in_gardening_community"`		         
	//ContractNumberAndDate *string  					`db:"contract_number_and_date"`  	     
	Parameters            json.RawMessage 	`json:"parameters"`
}


type Commercial struct {
	PropertyID          	  uuid.UUID		`json:"-"`
	IsNewCondition             *bool      	`json:"is_new_condition"`    
	PropertyType               *string		`json:"property_type"`         
	FloorNumber                *int8 		`json:"floor_number"`         
	BuildingFloors             *int8   		`json:"building_floors"`       
	TotalArea                  *float64		`json:"total_area"`        
	CommercialImprovements     []string 	`json:"commercial_improvements"`       
	CommercialRepair           *string    	`json:"commercial_repair"`     	
	PricePerSquareMeter        *float64		`json:"price_per_square_meter"`        
	RoomsRange                []int8        `json:"rooms_range"`
	CommercialBuildingLocation *string 		`json:"commercial_building_location"`        
	CommercialRentType		   *string		`json:"commercial_rent_type"`	   
	Parameters                 json.RawMessage `json:"parameters"`

	// IsPartlySellOrRent         *bool       
	// ContractNumberAndDate      *string           
}

type GarageAndParking struct {
	PropertyID          uuid.UUID				`json:"-"`
	PropertyType        *string					`json:"property_type"`  	 			    
	ParkingPlacesAmount *int16 		        	`json:"parking_places_amount"`
	TotalArea           *float64 	       		`json:"total_area"`
	Improvements        []string	      		`json:"improvements"`  
	Heating             *string 	      		`json:"heating"`  
	ParkingType         *string 	     		`json:"parking_type"`   
	Parameters          json.RawMessage 	`json:"parameters"`
}

type Room struct {
	PropertyID          	  uuid.UUID				`json:"-"`
	Condition                 *string				`json:"condition"`
	Bathroom                  *string				`json:"bathroom"`
	SuggestedRoomsAmount      *int16          		`json:"suggested_rooms_amount"`
	RoomsAmount               *int16            	`json:"rooms_amount"`
	FloorNumber               *int16            	`json:"floor_number"`
	BuildingFloors            *int16          		`json:"building_floors"`
	TotalArea                 *float64        		`json:"total_area"`
	IsBalcony                 *bool           		`json:"is_balcony"`
	RentalType                *string         		`json:"rental_type"`
	LivingSpaceArea           *float64        		`json:"living_space_area"`
	FlatRepair                *string         		`json:"flat_repair"`
	IsFurniture               *bool           		`json:"is_furniture"`
	KitchenSize               *float64        		`json:"kitchen_size"`
	KitchenItems              []string        		`json:"kitchen_items"`
	BathItems                 []string        		`json:"bath_items"`
	FlatRentForWhom           []string        		`json:"flat_rent_for_whom"`
	FlatWindowsSide           []string        		`json:"flat_windows_side"`
	YearBuilt                 *int16          		`json:"year_built"`
	WallMaterial              *string         		`json:"wall_material"`
	FlatImprovement           []string        		`json:"flat_improvement"`
	RoomType                  *string         		`json:"room_type"`
	ContractNumberAndDate     *string         		`json:"contract_number_and_date"`
	FlatBuildingImprovements  []string        		`json:"flat_building_improvements"`
	Parameters                json.RawMessage `json:"parameters"`
}



type Plot struct {
	PropertyID            uuid.UUID					`json:"-"`
	PlotArea              *float64  				`json:"plot_area"`      
	InGardeningCommunity  *bool     				`json:"in_gardening_community"`      
	PropertyRights        *string   				`json:"property_rights"`      
	Electricity           *string 					`json:"electricity"`        
	Water                 *string 					`json:"water"`        
	Gaz                   *string 					`json:"gaz"`        
	Sewage                *string  					`json:"sewage"`       
	IsOutbuildings        *bool    					`json:"is_outbuildings"`       
	OutbuildingsType      []string 					`json:"outbuildings_type"`       
	ContractNumberAndDate *string      				`json:"contract_number_and_date"`   
	Parameters            json.RawMessage	`json:"parameters"`	
}

type NewBuilding struct {
	PropertyID          uuid.UUID				`json:"-"`
	Deadline            *string         		`json:"deadline"`
	RoomOptions         []int16         		`json:"room_options"`
	Builder             *string         		`json:"builder"`
	ShareParticipation  *bool           		`json:"share_participation"`
	FloorOptions        []int16         		`json:"floor_options"`
	WallMaterial        *string         		`json:"wall_material"`
	CeilingHeight       *string         		`json:"flat_ceiling_height"`
	LayoutOptions       []string        		`json:"layout_options"`
	WithFinishing       *bool           		`json:"with_finishing"`
	Parameters          json.RawMessage	`json:"parameters"`
}