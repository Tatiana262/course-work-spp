package domain

import (
	"time"

	"github.com/google/uuid"
)

// FavoriteItem представляет собой одну запись о добавлении объекта в избранное.
type FavoriteItem struct {
	UserID    uuid.UUID
	MasterObjectID  uuid.UUID
	CreatedAt time.Time
}

// PaginatedFavoriteIDs - структура для ответа с пагинацией от репозитория.
type PaginatedFavoriteIDs struct {
	MasterObjectIDs []uuid.UUID
	TotalCount      int64 // Используем int64 для COUNT(*)
	CurrentPage  int
    ItemsPerPage int
}