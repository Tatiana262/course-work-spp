package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

)

// writeJSONError отправляет JSON-ответ с полем "error" и заданным статусом
func WriteJSONError(w http.ResponseWriter, statusCode int, message string) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(statusCode)

    // формируем объект ошибки
    json.NewEncoder(w).Encode(map[string]string{
        "error": message,
    })
}

// RespondWithJSON отправляет JSON-ответ
func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "Failed to marshal JSON response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func GetLimitOrDefault(r *http.Request) (*int, error) {
    limitStr := r.URL.Query().Get("limit")
	limit := 10 // дефолтное значение
	if limitStr != "" {	
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return nil, err
		}
	}
    return &limit, nil
}

func GetOffsetOrDefault(r *http.Request) (*int, error) {
    offsetStr := r.URL.Query().Get("offset")
	offset := 0
    if offsetStr != "" {
		var err error
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			return nil, err
		}
	}
    return &offset, nil;
}