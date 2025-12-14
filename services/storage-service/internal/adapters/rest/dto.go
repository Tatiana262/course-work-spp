package rest

import "time"

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
    PriceUSD float64   `json:"price_usd"`
    PriceBYN float64   `json:"price_byn"`
    Images   []string  `json:"images"`
    Address  string    `json:"address"`
    Status   string    `json:"status"`
    Category string    `json:"category"`
    DealType string    `json:"deal_type"`

    MasterObjectID	string    `json:"master_object_id"`
}

type ObjectGeneralInfoResponse struct {
    MasterObjectID	string    `json:"master_object_id"`
    ID       string    `json:"id"`
    Source   string    `json:"source"`
    SourceAdID int64  `json:"source_ad_id"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
    Category   string    `json:"category"` 
    AdLink     string    `json:"ad_link"` 
    SaleType   string   `json:"sale_type"` 
    Currency   string   `json:"currency"` 
    Images   []string  `json:"images"`
    ListTime  time.Time `json:"list_time"`
    Description string  `json:"description"`
    Title       string  `json:"title"`
    DealType string    `json:"deal_type"`
    CityOrDistrict string `json:"city_or_district"`
    Region      string  `json:"region"`    
    PriceUSD float64   `json:"price_usd"`
    PriceBYN float64   `json:"price_byn"`
    PriceEUR *float64   `json:"price_eur"`
    Address  string    `json:"address"`
    IsAgency bool      `json:"is_agency"` 
    SellerName string  `json:"seller_name"`     
    SellerDetails interface{} `json:"seller_details"`  
    Status   string    `json:"status"`
}

// PaginatedObjectsResponse - DTO для ответа со списком и пагинацией.
type PaginatedObjectsResponse struct {
    Data       []ObjectCardResponse `json:"objects"`
    Total      int                  `json:"total"`
    Page       int                  `json:"page"`
    PerPage    int                  `json:"per_page"`
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
    General ObjectGeneralInfoResponse `json:"general"` // Можно использовать ObjectCardResponse
    Details interface{} `json:"details"` // Сюда лягут детали (квартира, дом...)
    RelatedOffers []DuplicatesInfoResponse `json:"related_offers"`// Список всех других предложений
}


type GetByMasterIDsRequest struct {
    MasterIDs []string `json:"master_ids"`
}

type GetByMasterIDsResponse struct {
    Data []ObjectCardResponse `json:"data"`
}

// type FilterOptionsResponse map[string]FilterOptionResponse

type FilterResponse struct {
    Filters map[string]FilterOptionResponse `json:"filters"`
    Count   int                             `json:"count"`
}

type FilterOptionResponse struct {
    // Type    string        `json:"type"`
    Options []interface{} `json:"options,omitempty"`
    Min     interface{}   `json:"min,omitempty"`
    Max     interface{}   `json:"max,omitempty"`
}

type DictionaryItemsResponse map[string][]DictionaryItemResponse

type DictionaryItemResponse struct {
	SystemName  string `json:"system_name"`
	DisplayName string `json:"display_name"`
}