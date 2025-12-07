package domain

import (
	"time"

)

// PropertyLink представляет информацию о ссылке на объект недвижимости
type PropertyLink struct {
	ListedAt time.Time 
	Source   string    
	AdID int64 
	URL string 
}




// SearchCriteria определяет параметры для поиска в абстрактном виде.
// type SearchCriteria struct {
// 	PropertyType string
// 	DealType     string
// 	LocationUUID string // Для Realt.by это будет UUID города
// 	Page         int    // Realt.by использует пагинацию по страницам
// 	PageSize     int
// 	SortBy string
// 	SortOrder string
// }

type SearchCriteria struct {
	Name string
	LocationUUID   string
	Category       int    // Конкретный ID категории для API
	ObjectCategory []int  // Для новостроек
	ObjectType     []int  //Для коммерции, гаражей, машиномест
	Page           int

	//For debug
	Price  interface{}

	Rooms		[]int
}