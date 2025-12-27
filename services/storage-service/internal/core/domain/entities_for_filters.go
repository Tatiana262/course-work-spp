package domain


// type FilterOptions struct {
//     Category string
//     Region   string
//     DealType string
//     PriceCurrency string
// }


// FilterOption - описание одного фильтра для ответа
type FilterOption struct { 
    Options []interface{} 
    Min     interface{}   
    Max     interface{}   
}

type FilterOptionsResult struct {
	Options map[string]FilterOption
    Count int
}

type RangeResult struct { Min, Max interface{} }


// DictionaryItem - универсальная структура для элемента справочника
type DictionaryItem struct {
	SystemName  string 
	DisplayName string 
}