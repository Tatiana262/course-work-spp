package rest

import (
	"net/http"
	"storage-service/internal/core/domain"
	usecases_port "storage-service/internal/core/port/usecases_port"
	"strings"
)

type FilterHandler struct {
	getFilterOptionsUC usecases_port.GetFilterOptionsUseCase
	getDictionariesUC  usecases_port.GetDictionariesUseCase
}

func NewFilterHandler(getFilterOptionsUC usecases_port.GetFilterOptionsUseCase,
	getDictionariesUC  usecases_port.GetDictionariesUseCase) *FilterHandler {
	return &FilterHandler{
		getFilterOptionsUC:      getFilterOptionsUC,
		getDictionariesUC:		 getDictionariesUC,
	}
}

func (h *FilterHandler) GetFilterOptions(w http.ResponseWriter, r *http.Request) {
    // 1. Извлекаем query-параметры
    category := r.URL.Query().Get("category")
    if category == "" {
        WriteJSONError(w, http.StatusBadRequest, "Category is required")
        return
    }

    // 2. Собираем DTO для Use Case
    req := domain.FilterOptions{
        Category: category,
        Region:   r.URL.Query().Get("region"),
        DealType: r.URL.Query().Get("deal_type"),
    }
    
    // 3. Вызываем Use Case
    options, err := h.getFilterOptionsUC.Execute(r.Context(), req)
    if err != nil {
        WriteJSONError(w, http.StatusInternalServerError, "Failed to get filter options")
        return
    }

    response := make(FilterOptionsResponse)

    for key, value := range options {
        response[key] = FilterOptionResponse{
            Type: value.Type,
            Options: value.Options,
            Min: value.Min,
            Max: value.Max,
        }
    }

    // 4. Отправляем ответ
    RespondWithJSON(w, http.StatusOK, response)
}


func (h *FilterHandler) GetDictionaries(w http.ResponseWriter, r *http.Request) {
    // Получаем `names` из query-параметра
    namesStr := r.URL.Query().Get("names")
    var names []string
    if namesStr != "" {
        names = strings.Split(namesStr, ",")
    }

    // Вызываем Use Case. Если `names` пустой, он вернет все справочники.
    dictionaries, err := h.getDictionariesUC.Execute(r.Context(), names)
    if err != nil {
        // Use Case сам логирует ошибки, здесь просто возвращаем 500
        WriteJSONError(w, http.StatusInternalServerError, "Failed to retrieve dictionaries")
        return
    }

    response := make(DictionaryItemsResponse)
    for key, items := range dictionaries {
        for _, item := range items {
            responseItem := DictionaryItemResponse{
                SystemName: item.SystemName,
                DisplayName: item.DisplayName,
            }
            response[key] = append(response[key], responseItem)
        }
    }

    RespondWithJSON(w, http.StatusOK, response)
}