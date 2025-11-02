package rest

import "time"

// Структура для ответа API
type PropertyInfoResponse struct {
    ID        string    `json:"id"`
    SourceAdID int64     `json:"sourceAdId"`
    Source    string    `json:"source"`
    UpdatedAt time.Time `json:"updatedAt"`
}