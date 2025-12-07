package storage_api_client

type PropertyInfoResponse struct {
    // ID        string    `json:"id"`
    Source    string    `json:"source"`
    AdID      int64     `json:"ad_id"`
    AdLink      string  `json:"ad_url"`
    
    // UpdatedAt time.Time `json:"updatedAt"`
}
