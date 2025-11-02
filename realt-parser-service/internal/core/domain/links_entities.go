package domain

import "time"

// PropertyLink представляет информацию о ссылке на объект недвижимости
type PropertyLink struct {
	ListedAt time.Time `json:"listed_at"`
	Source   string    `json:"source"` // "realt"
	AdID int64 `json:"ad_id"`
	URL string `json:"ad_url"`
}


type SearchCriteria struct {
	Name string
	LocationUUID   string
	Category       int    // Конкретный ID категории для API
	ObjectCategory []int  
	ObjectType     []int  
	Page           int

	Rooms		[]int
}