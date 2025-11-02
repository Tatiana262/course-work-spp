package domain

import "time"

// PropertyLink представляет информацию о ссылке на объект недвижимости
type PropertyLink struct {
	ListedAt time.Time `json:"listed_at"`
	Source   string    `json:"source"`
	AdID int `json:"ad_id"`
}


type SearchCriteria struct {

	Category		string
	DealType    	string
	AdsAmount		int
	Location		string
	SortBy			string

	// Пагинация
	Cursor string 
}