package rest

// AddFavoriteRequest - тело запроса для добавления в избранное.
type AddFavoriteRequest struct {
	MasterObjectID string `json:"master_object_id"`
}

// ObjectCardResponse - структура для карточки объекта в ответе.
// Она должна соответствовать тому, что ожидает фронтенд.
type ObjectCardResponse struct {
	ID             string   `json:"id"`
	MasterObjectID string   `json:"master_object_id"`
	Title          string   `json:"title"`
	PriceUSD       float64  `json:"priceUSD"`
	Images         []string `json:"images"`
	Address        string   `json:"address"`
	Status         string   `json:"status"`
}

// PaginatedFavoritesResponse - структура для ответа со списком избранного.
type PaginatedFavoritesResponse struct {
	Data       []ObjectCardResponse `json:"data"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	PerPage    int                  `json:"perPage"`
}

// ErrorResponse - стандартная структура для ответа с ошибкой.
type ErrorResponse struct {
	Error string `json:"error"`
}