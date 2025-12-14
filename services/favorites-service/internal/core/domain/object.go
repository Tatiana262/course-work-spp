package domain

// ObjectCard - это представление объекта, которое мы получаем от storage-service.
// Это "доменное" представление внешних данных.
type ObjectCard struct {
	ID             string
	MasterObjectID string
	Title          string
	PriceUSD       float64
	PriceBYN       float64
	Images         []string
	Address        string
	Status         string
	Category 	   string
	DealType       string
}

// PaginatedObjectsResult - структура для финального ответа use case.
type PaginatedObjectsResult struct {
	Objects      []ObjectCard
	TotalCount   int64
	CurrentPage  int
	ItemsPerPage int
}