package realtfetcher

import "realt-parser-service/internal/core/domain"

// Константы, специфичные для адаптера
const (
	PageSize  = 50
	SortBy    = "updatedAt"
	SortOrder = "DESC"
)

// Структуры для GraphQL запроса
type Sort struct { 
	By string `json:"by"`; 
	Order string `json:"order"` 
}
type Pagination struct { 
	Page int `json:"page"`; 
	PageSize int `json:"pageSize"` 
}
type AddressV2 struct { 
	RegionUuid string `json:"stateRegionUuid"` 
}
type Where struct {
	AddressList      []AddressV2 `json:"addressV2"`
	Category         int         `json:"category"`
	ObjectCategory []int       `json:"objectCategory,omitempty"`
	ObjectType     []int		`json:"objectType,omitempty"`
	Rooms		   []int	   `json:"rooms,omitempty"`

	//debug
	Price          interface{} `json:"price,omitempty"`
}

type Data struct { 
	Where Where `json:"where"`; 
	Pagination Pagination `json:"pagination"`; 
	Sort []Sort `json:"sort"` 
}
type RequestVariables struct { 
	Data Data `json:"data"` 
}

// buildGraphQLVariables - теперь это простая функция-конструктор.
// Она не может вернуть ошибку.
func buildGraphQLVariables(criteria domain.SearchCriteria) RequestVariables {
	return RequestVariables{
		Data: Data{
			Where: Where{
				AddressList:      []AddressV2{{RegionUuid: criteria.LocationUUID}},
				Category:         criteria.Category,
				ObjectCategory:   criteria.ObjectCategory,
				ObjectType: 	  criteria.ObjectType,	
				Rooms: 			  criteria.Rooms,	
				
				//debug
				Price:            criteria.Price,
			},
			Pagination: Pagination{Page: criteria.Page, PageSize: PageSize},
			Sort:       []Sort{{By: SortBy, Order: SortOrder}},
		},
	}
}