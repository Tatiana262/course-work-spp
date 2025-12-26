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

    // Извлекаем query-параметры
    query := r.URL.Query()

    category := query.Get("category")
    if category == "" {
        WriteJSONError(w, http.StatusBadRequest, "Category is required")
        return
    }

    filters := domain.FindObjectsFilters{
		// Основные
		Category:       category,
		DealType:       parseString(query, "dealType"),
		PriceCurrency:  parseString(query, "priceCurrency"),
		PriceMin:       parseFloat(query, "priceMin"),
		PriceMax:       parseFloat(query, "priceMax"),
		Region:         parseString(query, "region"),
		CityOrDistrict: parseString(query, "cityOrDistrict"),
		Street:         parseString(query, "street"),

		// Общие для деталей
		Rooms:           parseIntSlice(query, "rooms"),
		TotalAreaMin:    parseFloat(query, "totalAreaMin"),
		TotalAreaMax:    parseFloat(query, "totalAreaMax"),
		LivingSpaceAreaMin: parseFloat(query, "livingSpaceAreaMin"),
		LivingSpaceAreaMax: parseFloat(query, "livingSpaceAreaMax"),
		KitchenAreaMin:  parseFloat(query, "kitchenAreaMin"),
		KitchenAreaMax:  parseFloat(query, "kitchenAreaMax"),
		YearBuiltMin:    parseInt(query, "yearBuiltMin"),
		YearBuiltMax:    parseInt(query, "yearBuiltMax"),
		WallMaterials:   parseStringSlice(query, "wallMaterials"),

		// Только для квартир
		FloorMin:         parseInt(query, "floorMin"),
		FloorMax:         parseInt(query, "floorMax"),
		FloorBuildingMin: parseInt(query, "floorBuildingMin"),
		FloorBuildingMax: parseInt(query, "floorBuildingMax"),
		RepairState:      parseStringSlice(query, "repairState"),
		BathroomType:     parseStringSlice(query, "bathroomType"),
		BalconyType:      parseStringSlice(query, "balconyType"),

		// Только для домов
		HouseTypes:        parseStringSlice(query, "houseTypes"),
		PlotAreaMin:       parseFloat(query, "plotAreaMin"),
		PlotAreaMax:       parseFloat(query, "plotAreaMax"),
		TotalFloors:       parseStringSlice(query, "totalFloors"),
		RoofMaterials:     parseStringSlice(query, "roofMaterials"),
		WaterConditions:   parseStringSlice(query, "waterConditions"),
		HeatingConditions: parseStringSlice(query, "heatingConditions"),
		ElectricityConditions: parseStringSlice(query, "electricityConditions"),
		SewageConditions:  parseStringSlice(query, "sewageConditions"),
		GazConditions:     parseStringSlice(query, "gazConditions"),

        // Для коммерции
        PropertyType: parseString(query, "commercialTypes"),
        CommercialImprovements: parseStringSlice(query, "commercialImprovements"),
        CommercialRepairs: parseStringSlice(query, "commercialRepairs"),
        CommercialLocation: parseStringSlice(query, "commercialBuildingLocations"),
        CommercialRoomsMin: parseInt(query, "roomsMin"),
        CommercialRoomsMax: parseInt(query, "roomsMax"),
	}
    
    // log.Println(filters.WallMaterials)
    // Вызываем Use Case
    result, err := h.getFilterOptionsUC.Execute(r.Context(), filters)
    if err != nil {
        WriteJSONError(w, http.StatusInternalServerError, "Failed to get filter options")
        return
    }

    responseFilters := make(map[string]FilterOptionResponse)

    for key, value := range result.Options {
        responseFilters[key] = FilterOptionResponse{
            // Type: value.Type,
            Options: value.Options,
            Min: value.Min,
            Max: value.Max,
        }
    }

    filterResponse := FilterResponse{
        Filters: responseFilters,
        Count: result.Count,
    }
    
    RespondWithJSON(w, http.StatusOK, filterResponse)
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