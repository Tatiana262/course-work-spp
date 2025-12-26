package storage_api_client

type PropertyInfoResponse struct {
    // ID        string    `json:"id"`
    Source    string    `json:"source"`
    AdID      int64     `json:"ad_id"`
    AdLink      string  `json:"ad_url"`
    
    // UpdatedAt time.Time `json:"updatedAt"`
}

type DictionaryItem struct {
    SystemName  string  `json:"system_name"`
    DisplayName string  `json:"display_name"`
}


type DictionaryItemsResponse map[string][]DictionaryItemResponse

type DictionaryItemResponse struct {
	SystemName  string `json:"system_name"`
	DisplayName string `json:"display_name"`
}