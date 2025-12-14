package rest

// AddFavoriteRequest - тело запроса для добавления в избранное.
type AddFavoriteRequest struct {
	MasterObjectID string `json:"master_object_id"`
}

// ObjectCardResponse - структура для карточки объекта в ответе.
// Она должна соответствовать тому, что ожидает фронтенд.

type ObjectCardResponse struct {
    ID       string    `json:"id"`
    Title  string    `json:"title"`
    PriceUSD float64   `json:"price_usd"`
    PriceBYN float64   `json:"price_byn"`
    Images   []string  `json:"images"`
    Address  string    `json:"address"`
    Status   string    `json:"status"`
    Category string    `json:"category"`
    DealType string    `json:"deal_type"`

    MasterObjectID	string    `json:"master_object_id"`
}

// PaginatedFavoritesResponse - структура для ответа со списком избранного.
type PaginatedFavoritesResponse struct {
	Data       []ObjectCardResponse `json:"data"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	PerPage    int                  `json:"per_page"`
}

// ErrorResponse - стандартная структура для ответа с ошибкой.
type ErrorResponse struct {
	Error string `json:"error"`
}