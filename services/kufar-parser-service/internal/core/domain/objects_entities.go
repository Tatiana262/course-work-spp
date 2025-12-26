package domain

import (
	"time"
	// "github.com/google/uuid"
)

const (
	StatusActive   = "active"
	StatusArchived = "archived"
)

// RealEstateRecord - это главная, агрегирующая структура для любого объекта недвижимости.
// Она объединяет общую часть (General) и специфичную (Details).
type RealEstateRecord struct {
    General GeneralProperty
    Details interface{} // Сюда будет помещен указатель на Apartment, House, Commercial и т.д.
}

// GeneralProperty представляет основную, общую информацию для любого объекта недвижимости.
// Соответствует таблице `general_properties`.
type GeneralProperty struct {
	Source       	 string    	
	SourceAdID   	 int64     	
	// Category         string    
	AdLink           string	   
	RemunerationType string
	Currency         string	     
	Images           []string   
	ListTime         time.Time	 
	Body             string  	   
	Subject          string		     
	DealType         string		     
	Latitude         float64	
	Longitude        float64	
	CityOrDistrict   string		
	Region           string		
	PriceBYN         float64	
	PriceUSD         float64	
	PriceEUR         *float64	
	Address        	 string 		

	IsAgency         bool 	
	SellerName     string  	
	SellerDetails map[string]interface{} 	

	Status string 
	
	// ContactPerson  *string 		 
	// UNPNumber      *string  	
	// CompanyAddress *string  	
	// CompanyLicense  *string		  
	// ImportLink     *string  	
}

// --- Структуры для специализированных данных ---

// Apartment представляет детали для квартир.
// Соответствует таблице `apartments`.
type Apartment struct {
	RoomsAmount             *int8 	
	FloorNumber             *int8 		
	BuildingFloors          *int8  
	TotalArea               *float64  
	LivingSpaceArea         *float64 	
	KitchenArea             *float64 	
	YearBuilt               *int16   
	WallMaterial            *string 
	RepairState             *string 	
	BathroomType            *string 	 
	Balcony                 *string 	
	PricePerSquareMeter     *float64 			  		

	IsNewCondition                     *bool   	
	// ContractNumberAndDate   *string 	        
	Parameters              map[string]interface{} 
}

// House представляет детали для дома, дачи.
// Соответствует таблице `houses`.
type House struct {
	TotalArea             *float64    	    
	PlotArea              *float64    	    
	WallMaterial          *string 		              
	YearBuilt             *int16          
	LivingSpaceArea       *float64     	  
	BuildingFloors        *int8          
	RoomsAmount           *int8		         
	KitchenArea           *float64    	    
	Electricity           *string 			         		         
	Water                 *string        
	Heating               *string   	      
	Sewage                *string     	    
	Gaz                   *string       
	RoofMaterial          *string    	     	    	     
	HouseType             *string  		
	
	CompletionPercent 	  *int8   // Kufar: house_readiness

	Parameters            map[string]interface{} 

	IsNewCondition                     *bool  
	// InGardeningCommunity  *bool  
	// ContractNumberAndDate *string
}


// Commercial представляет детали для коммерческой недвижимости.
// Соответствует таблице `commercial`.
type Commercial struct {
	IsNewCondition             *bool          
	PropertyType               *string         
	FloorNumber                *int8          
	BuildingFloors             *int8          
	TotalArea                  *float64        
	CommercialImprovements     []string        
	CommercialRepair           *string         	
	PricePerSquareMeter        *float64        
	RoomsRange                []int8          
	CommercialBuildingLocation *string         
	CommercialRentType		   *string		   
	Parameters                 map[string]interface{} 

	// IsPartlySellOrRent         *bool       
	// ContractNumberAndDate      *string           
}




// GarageAndParking представляет детали для гаражей и стоянок.
// Соответствует таблице `garages_and_parkings`.
type GarageAndParking struct { 
	PropertyType        *string    	//     
	ParkingPlacesAmount *int16 		//        
	TotalArea           *float64 	//       
	Improvements        []string	//        
	Heating             *string 	//        
	ParkingType         *string 	//        
	Parameters          map[string]interface{} 
}

// Room представляет детали для комнат.
// Соответствует таблице `rooms`.
type Room struct {
	Condition                 *string         //
	Bathroom                  *string         //
	SuggestedRoomsAmount      *int16          //
	RoomsAmount               *int16          //
	FloorNumber               *int16          //
	BuildingFloors            *int16          //
	TotalArea                 *float64        //
	IsBalcony                 *bool           //
	RentalType                *string         //
	LivingSpaceArea           *float64        //
	FlatRepair                *string         //
	IsFurniture               *bool           //
	KitchenSize               *float64        //
	KitchenItems              []string        //
	BathItems                 []string        //
	FlatRentForWhom           []string        //
	FlatWindowsSide           []string        //
	YearBuilt                 *int16          //
	WallMaterial              *string         //
	FlatImprovement           []string        //
	RoomType                  *string         //
	ContractNumberAndDate     *string         //
	FlatBuildingImprovements  []string        //
	Parameters                map[string]interface{}
}




// Plot представляет детали для земельных участков.
// Соответствует таблице `plots`.
type Plot struct {
	PlotArea              *float64        //
	InGardeningCommunity  *bool           //
	PropertyRights        *string         //
	Electricity           *string         //
	Water                 *string         //
	Gaz                   *string         //
	Sewage                *string         //
	IsOutbuildings        *bool           //
	OutbuildingsType      []string        //
	ContractNumberAndDate *string         //
	Parameters            map[string]interface{}
}


// NewBuilding представляет детали для новостроек.
// Соответствует таблице `new_buildings`.
type NewBuilding struct {
	Deadline            *string         
	RoomOptions         []int16         
	Builder             *string         
	ShareParticipation  *bool           
	FloorOptions        []int16         
	WallMaterial        *string         
	CeilingHeight       *string         
	LayoutOptions       []string        
	WithFinishing       *bool           
	Parameters          map[string]interface{} 
}



// type PropertyRecord struct {
// 	ID                   string           `json:"-" db:"id"`
// 	CreatedAt            time.Time        `json:"-" db:"created_at"`
// 	UpdatedAt            *time.Time       `json:"-" db:"updated_at"`
// 	DeletedAt            *time.Time       `json:"-" db:"deleted_at"`
// 	IsDuplicated         bool             `json:"-" db:"is_duplicated"`
// 	CreatedByID          *string          `json:"-" db:"created_by_id"`
// 	PublishedByID        *string          `json:"-" db:"published_by_id"`
// 	ResponsibleManagerID *string          `json:"-" db:"responsible_manager_id"`
	
// 	Partner       string           `json:"partner" db:"partner"`
// 	Source        string           `json:"source" db:"source"`
// 	Slug          string           `json:"slug" db:"slug"`
// 	Title         string           `json:"title" db:"title"`
// 	Description   *string          `json:"description,omitempty" db:"description"`
// 	Images        []string         `json:"images,omitempty" db:"images"`
// 	PreviewImage  *string          `json:"preview_image,omitempty" db:"preview_image"`
// 	Address       string           `json:"address" db:"address"`
// 	District      *string          `json:"district,omitempty" db:"district"`
// 	Region        *string          `json:"region,omitempty" db:"region"`
// 	Coordinates   *json.RawMessage `json:"coordinates,omitempty" db:"coordinates"`
// 	Contacts      *json.RawMessage `json:"contacts,omitempty" db:"contacts"`

// 	TransactionType *string `json:"transaction_type,omitempty" db:"transaction_type"`
// 	Estate          *string `json:"estate,omitempty" db:"estate"`
// 	AdvertType      *string `json:"advert_type,omitempty" db:"advert_type"`
// 	AdvertiserType  *string `json:"advertiser_type,omitempty" db:"advertiser_type"`
// 	Status          *string `json:"status" db:"status"`

// 	TotalPrice          *float64 `json:"total_price,omitempty" db:"total_price"`
// 	RentPrice           *float64 `json:"rent_price,omitempty" db:"rent_price"`
// 	DepositPrice        *float64 `json:"deposit_price,omitempty" db:"deposit_price"`
// 	PricePerSquareMeter *float64  `json:"price_per_square_meter,omitempty" db:"price_per_square_meter"` 
// 	Currency            *string  `json:"currency,omitempty" db:"currency"`
	
// 	AreaInSquareMeters *float64 `json:"area_in_square_meters,omitempty" db:"area_in_square_meters"`
// 	RoomsNumString     *string  `json:"rooms_number_string,omitempty" db:"rooms_number_string"`
// 	RoomsNum           []string `json:"rooms_num,omitempty" db:"rooms_num"`
// 	FloorNumber        *int     `json:"floor_number,omitempty" db:"floor_number"`
// 	BuildingFloors     *int     `json:"building_floors,omitempty" db:"building_floors"`
// 	Note               *string  `json:"note,omitempty" db:"note"`
// 	Features           []string `json:"features,omitempty" db:"features"`
// 	ParsedFeatures     []string `json:"parsed_features,omitempty" db:"parsed_features"`
// 	Marks              []string `json:"marks,omitempty" db:"marks"`

// 	PublishedAt   *time.Time `json:"published_at,omitempty" db:"published_at"`
// 	SiteCreatedAt *time.Time `json:"site_created_at" db:"site_created_at"`
// 	SiteUpdatedAt *time.Time `json:"site_updated_at" db:"site_updated_at"`
// }