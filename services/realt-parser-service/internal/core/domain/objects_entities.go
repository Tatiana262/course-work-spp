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
	Source       	 string    			//realt
	SourceAdID   	 int64     			//code
	// Category         string    		//category
	AdLink           string	
	SaleType string		 // можно termsOfSale
	Currency         string	     	// можно вытянуть из массива объектов "normalizedPriceHistory" из одного из объектов по полю "priceCurrency"
	Images           []string   	// slides
	ListTime         time.Time	 	//createdAt
	Description             string  		// description	   
	Title          string			// headline	     
	DealType         string		     // скорее всего просто по полям, специфичным для продажи/аренды
	Latitude         float64	
	Longitude        float64	
	CityOrDistrict   string			// stateDistrictName + townName или просто townName
	Region           string		    // stateRegionName
	PriceBYN         float64	
	PriceUSD         float64	
	PriceEUR         *float64		// наверное надо сделать обязательным, так как в основном есть 
	Address        	 string 		// address

	IsAgency         bool 	  	// смотря есть объект agency или нет (бывший CompanyAd)
	SellerName     string  		// contactName либо просто название агенства из объекта "agency" поле "title", но есть ещё agent
	SellerDetails map[string]interface{} 

	Status string
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
	BalconyType             *string 
	PricePerSquareMeter     *float64 	

	//Condition               *string   	// непонятно, что с этим делать (скорее всего в параметры)     
	//ContractNumberAndDate   *string     // agencyContract.contract надо вынести в параметры  
	Parameters              map[string]interface{} 
}


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
	
	CompletionPercent 	  *int8  

	Parameters            map[string]interface{} 

	// Condition             *string 
	// InGardeningCommunity  *bool  
	// ContractNumberAndDate *string
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

// Commercial представляет детали для коммерческой недвижимости.
// Соответствует таблице `commercial`.
type Commercial struct {
	Condition                  *string         //
	PropertyType               *string         //
	FloorNumber                *int16          //
	BuildingFloors             *int16          //
	TotalArea                  *float64        //
	CommercialImprovements     []string        //
	CommercialRepair           *string         //
	IsPartlySellOrRent         *bool         //
	PricePerSquareMeter        *float64        //
	ContractNumberAndDate      *string         //
	RoomsAmount                *int16          //
	CommercialBuildingLocation *string         //
	CommercialRentType		   *string		   //
	Parameters                 map[string]interface{} 
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
