package rest

import (
	"encoding/json"
	"net/http"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"
	"storage-service/internal/core/port/usecases_port"
	"strconv"

	// "strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type GetInfoHandler struct {
	findObjectsUC      usecases_port.FindObjectsUseCase
	getObjectDetailsUC usecases_port.GetObjectDetailsUseCase
	getBestObjectsUC   usecases_port.GetBestObjectsByMasterIDsUseCase
}


func NewGetInfoHandler(findObjectsUC usecases_port.FindObjectsUseCase, 
	getObjectDetailsUC usecases_port.GetObjectDetailsUseCase,
	getBestObjectsUC   usecases_port.GetBestObjectsByMasterIDsUseCase) *GetInfoHandler {
		return &GetInfoHandler{
			findObjectsUC: findObjectsUC,
			getObjectDetailsUC: getObjectDetailsUC,
			getBestObjectsUC:  getBestObjectsUC,
		}
}

// --- НОВЫЕ ОБРАБОТЧИКИ ---

// FindObjects обрабатывает GET /api/v1/objects
func (h *GetInfoHandler) FindObjects(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context())

	// --- Шаг 1: Парсим query-параметры ---
	query := r.URL.Query()

	// --- Шаг 1: Парсим пагинацию ---
	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(query.Get("perPage"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	limit := perPage
	offset := (page - 1) * perPage

	// --- Шаг 2: Собираем фильтры с помощью наших хелперов ---
	filters := domain.FindObjectsFilters{
		// Основные
		Category:       parseString(query, "category"),
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

	handlerLogger := logger.WithFields(port.Fields{
		"handler": "FindObjects",
		"page":    page,
		"per_page": perPage,
		"filters": filters, // `filters` будет красиво сериализован в JSON
	})
	handlerLogger.Debug("Processing request to find objects", nil)

	// --- Шаг 3: Вызываем use-case ---
	paginatedResult, err := h.findObjectsUC.Execute(r.Context(), filters, limit, offset)
	if err != nil {
		handlerLogger.Error("Use case failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to retrieve objects")
		return
	}

	handlerLogger.Info("Successfully found objects", port.Fields{
		"total_found": paginatedResult.TotalCount,
		"items_on_page": len(paginatedResult.Objects),
	})

	// --- Шаг 4: Маппим результат в DTO для ответа ---
	response := PaginatedObjectsResponse{
		Total:      paginatedResult.TotalCount,
		Page:       paginatedResult.CurrentPage,
		PerPage:    paginatedResult.ItemsPerPage,
		Data:       make([]ObjectCardResponse, len(paginatedResult.Objects)),
	}

	for i, obj := range paginatedResult.Objects {
		response.Data[i] = ObjectCardResponse{
			ID:       obj.ID.String(),
			Title:  obj.Title, // Предполагаем, что у вас есть поле Title
			PriceUSD: obj.PriceUSD,
			PriceBYN: obj.PriceBYN,
			Images:   obj.Images,
			Address:  obj.Address,
			Status:   obj.Status,
			MasterObjectID: obj.MasterObjectID,
			Category: obj.Category,
			DealType: obj.DealType,
		}
	}

	// --- Шаг 5: Отправляем JSON ---
	RespondWithJSON(w, http.StatusOK, response)
}

// GetObjectDetails обрабатывает GET /api/v1/objects/{objectID}
func (h *GetInfoHandler) GetObjectDetails(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context())

	// --- Шаг 1 & 2: Получаем и парсим objectID из URL ---
	objectIDStr := chi.URLParam(r, "objectID")
	objectID, err := uuid.Parse(objectIDStr)
	if err != nil {
		logger.Warn("Invalid object ID format", port.Fields{"error": err.Error()})
		WriteJSONError(w, http.StatusBadRequest, "Invalid object ID format")
		return
	}

	handlerLogger := logger.WithFields(port.Fields{
		"handler": "GetObjectDetails",
		"object_id":    objectIDStr,
	})
	handlerLogger.Debug("Processing request to find object details", nil)

	// --- Шаг 3: Вызываем use-case ---
	detailsView, err := h.getObjectDetailsUC.Execute(r.Context(), objectID)
	if err != nil {
		handlerLogger.Error("Use case failed", err, nil)
		WriteJSONError(w, http.StatusNotFound, "Object not found")
		return
	}

	// --- Шаг 4: Маппим результат в DTO для ответа ---
	// Маппим основную информацию
	generalResponse := ObjectGeneralInfoResponse{
		MasterObjectID: detailsView.MainProperty.MasterObjectID,		
		ID:       detailsView.MainProperty.ID.String(),
		Source: detailsView.MainProperty.Source,
		SourceAdID: detailsView.MainProperty.SourceAdID,
		CreatedAt: detailsView.MainProperty.CreatedAt,
		UpdatedAt: detailsView.MainProperty.UpdatedAt,
		Category: detailsView.MainProperty.Category,
		AdLink: detailsView.MainProperty.AdLink,
		SaleType: detailsView.MainProperty.SaleType,
		Currency: detailsView.MainProperty.Currency,
		Images:   detailsView.MainProperty.Images,
		ListTime: detailsView.MainProperty.ListTime,
		Description: detailsView.MainProperty.Description,
		Title:  detailsView.MainProperty.Title,
		DealType: detailsView.MainProperty.DealType,
		CityOrDistrict: detailsView.MainProperty.CityOrDistrict,
		Region: detailsView.MainProperty.Region,
		PriceUSD: detailsView.MainProperty.PriceUSD,
		PriceBYN: detailsView.MainProperty.PriceBYN,
		PriceEUR: detailsView.MainProperty.PriceEUR,
		Address:  detailsView.MainProperty.Address,
		IsAgency: detailsView.MainProperty.IsAgency,
		SellerName: detailsView.MainProperty.SellerName,
		SellerDetails: detailsView.MainProperty.SellerDetails,
		Status:   detailsView.MainProperty.Status,		
	}

	// Маппим детали и связанные предложения
	response := ObjectDetailsResponse{
		General: generalResponse,
		Details: detailsView.Details, // Детали уже в нужном формате (interface{})
		RelatedOffers: make([]DuplicatesInfoResponse, len(detailsView.RelatedOffers)),
	}

	for i, offer := range detailsView.RelatedOffers {
		response.RelatedOffers[i] = DuplicatesInfoResponse{
			ID:       offer.ID.String(),
			Source:   offer.Source,
			AdLink:   offer.AdLink,
			IsSourceDuplicate: offer.IsSourceDuplicate,
			DealType: offer.DealType,
		}
	}

	handlerLogger.Info("Successfully found object details", nil)
	// --- Шаг 5: Отправляем JSON ---
	RespondWithJSON(w, http.StatusOK, response)
}

// для избранного
func (h *GetInfoHandler) GetBestByMasterIDs(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context())

    var req GetByMasterIDsRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid request body", port.Fields{"error": err.Error()})
        WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    if len(req.MasterIDs) > 100 { // Ограничение, чтобы не перегружать сервис
		logger.Warn("Too many MasterIDs requested", port.Fields{"max": 100})
        WriteJSONError(w, http.StatusBadRequest, "Too many IDs requested, max 100")
        return
    }

	handlerLogger := logger.WithFields(port.Fields{
		"handler": "GetBestByMasterIDs",
		"ids_amount":  len(req.MasterIDs),
	})
	handlerLogger.Debug("Processing request to find objects by master ids", nil)

    // Вызываем Use Case, который вызовет наш новый метод репозитория
    objects, err := h.getBestObjectsUC.Execute(r.Context(), req.MasterIDs)
    if err != nil {
		handlerLogger.Error("Use case failed", err, nil)
        WriteJSONError(w, http.StatusInternalServerError, "Failed to fetch objects")
        return
    }

	response := GetByMasterIDsResponse{
		Data:       make([]ObjectCardResponse, len(objects)),
	}

	for i, obj := range objects {
		response.Data[i] = ObjectCardResponse{
			ID:       obj.ID.String(),
			Title:  obj.Title, // Предполагаем, что у вас есть поле Title
			PriceUSD: obj.PriceUSD,
			PriceBYN: obj.PriceBYN,
			Images:   obj.Images,
			Address:  obj.Address,
			Status:   obj.Status,
			MasterObjectID: obj.MasterObjectID,
			Category: obj.Category,
			DealType: obj.DealType,
		}
	}

	handlerLogger.Info("Successfully found objects by master ids", port.Fields{
		"total_found": len(objects),
	})

	RespondWithJSON(w, http.StatusOK, response)
}