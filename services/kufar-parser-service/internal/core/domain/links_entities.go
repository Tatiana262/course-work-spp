package domain

import "time"

// PropertyLink представляет информацию о ссылке на объект недвижимости
type PropertyLink struct {
	
	ListedAt time.Time 
	Source   string     
	AdID int64 
	
}

// Criteria определяет параметры для поиска ссылок на недвижимость
type SearchCriteria struct {

	Name 			string

	Category		string
	DealType    	string
	AdsAmount		int
	Location		string
	SortBy			string

	Query			string
	// Пагинация
	Cursor string 
}