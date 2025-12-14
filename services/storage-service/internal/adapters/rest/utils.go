package rest

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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




// parseString извлекает строковый параметр.
func parseString(q url.Values, key string) string {
	return q.Get(key)
}

// parseFloat извлекает float64 параметр.
func parseFloat(q url.Values, key string) *float64 {
	strVal := q.Get(key)
	if strVal == "" {
		return nil
	}
	if val, err := strconv.ParseFloat(strVal, 64); err == nil {
		return &val
	}
	return nil
}

// parseInt извлекает int параметр.
func parseInt(q url.Values, key string) *int {
	strVal := q.Get(key)
	if strVal == "" {
		return nil
	}
	if val, err := strconv.Atoi(strVal); err == nil {
		return &val
	}
	return nil
}

// parseIntSlice извлекает срез int, разделенных запятыми (например, "1,2,5").
func parseIntSlice(q url.Values, key string) []int {
	strVal := q.Get(key)
	if strVal == "" {
		return nil
	}
	parts := strings.Split(strVal, ",")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		if val, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			result = append(result, val)
		}
	}
	return result
}

// parseStringSlice извлекает срез строк, разделенных запятыми.
func parseStringSlice(q url.Values, key string) []string {
	strVal := q.Get(key)
	if strVal == "" {
		return nil
	}
	log.Println(strVal)
	parts := strings.Split(strVal, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}