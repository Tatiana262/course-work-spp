package domain

// FilterOptionsRequest - параметры для запроса опций.
type FilterOptions struct {
    Category string
    Region   string
    DealType string
}


// FilterOption - описание одного фильтра для ответа.
type FilterOption struct {
    Type    string        
    Options []interface{} 
    Min     interface{}   
    Max     interface{}   
}

type RangeResult struct { Min, Max float64 }


// DictionaryItem - универсальная структура для элемента справочника.
type DictionaryItem struct {
	SystemName  string 
	DisplayName string 
}