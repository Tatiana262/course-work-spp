package domain

import (
	"github.com/google/uuid"
)

// ObjectCard - это представление объекта, которое мы получаем от storage-service.
// Это "доменное" представление внешних данных.
type ObjectCard struct {
	ID             uuid.UUID
	MasterObjectID uuid.UUID
	Title          string
	PriceUSD       float64
	Images         []string
	Address        string
	Status         string
}

// PaginatedObjectsResult - структура для финального ответа use case.
type PaginatedObjectsResult struct {
	Objects      []ObjectCard
	TotalCount   int64
	CurrentPage  int
	ItemsPerPage int
}