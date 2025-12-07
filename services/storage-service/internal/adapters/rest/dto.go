package rest

// import "time"

// Структура для ответа API
type PropertyInfoResponse struct {
    // ID        string    `json:"id"`
    Source    string    `json:"source"`
    AdID      int64     `json:"ad_id"`
    AdLink      string  `json:"ad_url"`
    
    // UpdatedAt time.Time `json:"updatedAt"`
}


// ObjectCardResponse - DTO для карточки объекта в списке.
type ObjectCardResponse struct {
    ID       string    `json:"id"`
    Title  string    `json:"title"`
    PriceUSD float64   `json:"priceUSD"`
    Images   []string  `json:"images"`
    Address  string    `json:"address"`
    Status   string    `json:"status"`

    MasterObjectID	string    `json:"master_object_id"`
    // ... другие поля для карточки
}

// PaginatedObjectsResponse - DTO для ответа со списком и пагинацией.
type PaginatedObjectsResponse struct {
    Data       []ObjectCardResponse `json:"data"`
    Total      int                  `json:"total"`
    Page       int                  `json:"page"`
    PerPage    int                  `json:"perPage"`
}

type DuplicatesInfoResponse struct {
	ID           string    `json:"id"`		
	Source       string    `json:"source"`		    			 
	AdLink       string    `json:"ad_link"`
	IsSourceDuplicate bool `json:"is_source_duplicate"`
    DealType	 string    `json:"deal_type"`
}

// ObjectDetailsResponse - DTO для детальной страницы.
type ObjectDetailsResponse struct {
    General ObjectCardResponse `json:"general"` // Можно использовать ObjectCardResponse
    Details interface{} `json:"details"` // Сюда лягут детали (квартира, дом...)
    RelatedOffers []DuplicatesInfoResponse // Список всех других предложений
}


type GetByMasterIDsRequest struct {
    MasterIDs []string `json:"master_ids"`
}

type GetByMasterIDsResponse struct {
    Data []ObjectCardResponse `json:"data"`
}

type FilterOptionsResponse map[string]FilterOptionResponse

type FilterOptionResponse struct {
    Type    string        `json:"type"`
    Options []interface{} `json:"options,omitempty"`
    Min     interface{}   `json:"min,omitempty"`
    Max     interface{}   `json:"max,omitempty"`
}

type DictionaryItemsResponse map[string][]DictionaryItemResponse

type DictionaryItemResponse struct {
	SystemName  string `json:"systemName"`
	DisplayName string `json:"displayName"`
}