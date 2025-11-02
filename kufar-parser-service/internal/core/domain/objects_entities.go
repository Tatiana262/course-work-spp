package domain

import (
	"time"
)

// RealEstateRecord - это главная, агрегирующая структура для объекта недвижимости
type RealEstateRecord struct {
    General GeneralProperty
    Details interface{} 
}

// основная информацию для любого объекта недвижимости
type GeneralProperty struct {
	Source       	 string    	
	SourceAdID   	 int64     	
	Category         string    
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
	
}


// детали для квартир
type Apartment struct {
	RoomsAmount             *int16 	
	FloorNumber             *int16 		
	BuildingFloors          *int16  
	TotalArea               *float64  
	LivingSpaceArea         *float64 	
	KitchenArea             *float64 	
	YearBuilt               *int16   
	WallMaterial            *string 
	RepairState             *string 	
	BathroomType            *string 	 
	Balcony                 *string 	
	PricePerSquareMeter     *float64 			  		

	// Condition               *string   	
	// ContractNumberAndDate   *string 	        
	Parameters              map[string]interface{} 
}

// детали для дома, дачи
type House struct {
	TotalArea             *float64    	    
	PlotArea              *float64    	    
	WallMaterial          *string 		       
	Condition             *string        
	YearBuilt             *int16          
	LivingSpaceArea       *float64     	  
	BuildingFloors        *int16          
	RoomsAmount           *int16		         
	KitchenSize           *float64    	    
	Electricity           *bool			         
	InGardeningCommunity  *bool  		         
	Water                 *string        
	Heating               *string   	      
	Sewage                *string     	    
	Gaz                   *string       
	RoofMaterial          *string    	     
	ContractNumberAndDate *string    	     
	HouseType             *string  		      
	Parameters            map[string]interface{} 
}

// детали для гаражей и стоянок.
type GarageAndParking struct { 
	PropertyType        *string    	     
	ParkingPlacesAmount *int16 		        
	TotalArea           *float64 	       
	Improvements        []string	        
	Heating             *string 	        
	ParkingType         *string 	        
	Parameters          map[string]interface{} 
}

// детали для комнат
type Room struct {
	Condition                 *string         
	Bathroom                  *string         
	SuggestedRoomsAmount      *int16          
	RoomsAmount               *int16          
	FloorNumber               *int16          
	BuildingFloors            *int16          
	TotalArea                 *float64        
	IsBalcony                 *bool           
	RentalType                *string         
	LivingSpaceArea           *float64        
	FlatRepair                *string         
	IsFurniture               *bool           
	KitchenSize               *float64        
	KitchenItems              []string        
	BathItems                 []string        
	FlatRentForWhom           []string        
	FlatWindowsSide           []string        
	YearBuilt                 *int16          
	WallMaterial              *string         
	FlatImprovement           []string        
	RoomType                  *string         
	ContractNumberAndDate     *string         
	FlatBuildingImprovements  []string        
	Parameters                map[string]interface{}
}

// детали для коммерческой недвижимости
type Commercial struct {
	Condition                  *string         
	PropertyType               *string         
	FloorNumber                *int16          
	BuildingFloors             *int16          
	TotalArea                  *float64        
	CommercialImprovements     []string        
	CommercialRepair           *string         
	IsPartlySellOrRent         *bool         
	PricePerSquareMeter        *float64        
	ContractNumberAndDate      *string         
	RoomsAmount                *int16          
	CommercialBuildingLocation *string         
	CommercialRentType		   *string		   
	Parameters                 map[string]interface{} 
}


// детали для земельных участков
type Plot struct {
	PlotArea              *float64        
	InGardeningCommunity  *bool           
	PropertyRights        *string         
	Electricity           *string         
	Water                 *string         
	Gaz                   *string         
	Sewage                *string         
	IsOutbuildings        *bool           
	OutbuildingsType      []string        
	ContractNumberAndDate *string         
	Parameters            map[string]interface{}
}


// детали для новостроек
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