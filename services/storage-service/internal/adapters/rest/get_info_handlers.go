package rest

import (
	"encoding/json"
	// "log"
	"net/http"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"
	"storage-service/internal/core/port/usecases_port"
	"strconv"
	"strings"

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

	// Пагинация с значениями по умолчанию
	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(query.Get("perPage"))
	if perPage < 1 || perPage > 100 { // Ограничиваем максимальное количество
		perPage = 20
	}
	limit := perPage
	offset := (page - 1) * perPage

	// --- Шаг 2: Собираем фильтры ---
	filters := domain.FindObjectsFilters{
		Category: query.Get("category"),
		Region:   query.Get("region"),
		DealType: query.Get("dealType"),
	}

	// Парсим цену
	if priceMinStr := query.Get("priceMin"); priceMinStr != "" {
		if priceMin, err := strconv.ParseFloat(priceMinStr, 64); err == nil {
			filters.PriceMin = &priceMin
		}
	}
	if priceMaxStr := query.Get("priceMax"); priceMaxStr != "" {
		if priceMax, err := strconv.ParseFloat(priceMaxStr, 64); err == nil {
			filters.PriceMax = &priceMax
		}
	}
    
    // Парсим комнаты (например, ?rooms=1,2,3)
	if roomsStr := query.Get("rooms"); roomsStr != "" {
		roomsParts := strings.Split(roomsStr, ",")
		rooms := make([]int, 0, len(roomsParts))
		for _, part := range roomsParts {
			if room, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
				rooms = append(rooms, room)
			}
		}
		if len(rooms) > 0 {
			filters.Rooms = rooms
		}
	}

	handlerLogger := logger.WithFields(port.Fields{
		"handler": "FindObjects",
		"page":    page,
		"per_page": perPage,
		"filters": filters, // `filters` будет красиво сериализован в JSON
	})
	handlerLogger.Info("Processing request to find objects", nil)

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
			Images:   obj.Images,
			Address:  obj.Address,
			Status:   obj.Status,
			MasterObjectID: obj.MasterObjectID,
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
	handlerLogger.Info("Processing request to find object details", nil)

	// --- Шаг 3: Вызываем use-case ---
	detailsView, err := h.getObjectDetailsUC.Execute(r.Context(), objectID)
	if err != nil {
		handlerLogger.Error("Use case failed", err, nil)
		WriteJSONError(w, http.StatusNotFound, "Object not found")
		return
	}

	// --- Шаг 4: Маппим результат в DTO для ответа ---
	// Маппим основную информацию
	generalResponse := ObjectCardResponse{
		ID:       detailsView.MainProperty.ID.String(),
		Title:  detailsView.MainProperty.Title,
		PriceUSD: detailsView.MainProperty.PriceUSD,
		Images:   detailsView.MainProperty.Images,
		Address:  detailsView.MainProperty.Address,
		Status:   detailsView.MainProperty.Status,
		MasterObjectID: detailsView.MainProperty.MasterObjectID,
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
	handlerLogger.Info("Processing request to find objects by master ids", nil)

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
			Images:   obj.Images,
			Address:  obj.Address,
			Status:   obj.Status,
			MasterObjectID: obj.MasterObjectID,
		}
	}

	RespondWithJSON(w, http.StatusOK, response)
}