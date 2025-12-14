package storage_api_client

import "github.com/google/uuid"

// DTO для запроса к storage-service
type getByMasterIDsRequest struct {
	MasterIDs []uuid.UUID `json:"master_ids"`
}

// DTO для ответа от storage-service
// Эта структура должна в точности совпадать с `ObjectCardResponse` из storage-service
type objectCardResponse struct {
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

type getByMasterIDsResponse struct {
    Data []objectCardResponse `json:"data"`
}