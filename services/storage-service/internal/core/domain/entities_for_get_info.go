package domain

import (
	"time"

	"github.com/google/uuid"
)

// FindObjectsFilters - структура для передачи всех возможных фильтров.
type FindObjectsFilters struct {
    Category     string
    Region       string
	DealType     string   
    Rooms        []int
    PriceMin     *float64
    PriceMax     *float64
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
	ID           uuid.UUID 			
	Source       string    			
	SourceAdID   int64     			
	UpdatedAt    time.Time 			
	Category         string    
	DealType         string     	
	AdLink           string     	
	Title          	string     		
	Address        	string 			
	
	PriceBYN         float64   		
	PriceUSD         float64   		
	PriceEUR         *float64   	
	Currency         string     	
	
	Images           []string   	

	OffersCount  int // <--- НОВОЕ ПОЛЕ
	Status    		string

	MasterObjectID	string		
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