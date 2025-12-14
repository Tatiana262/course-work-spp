package rest

import (
	"encoding/json"
	"favorites-service/internal/contextkeys"
	"favorites-service/internal/core/port"
	"favorites-service/internal/core/port/usecases_port"
	// "log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// FavoritesHandler реализует интерфейс Handlers.
type FavoritesHandler struct {
	addUC    usecases_port.AddToFavoritesUseCasePort
	removeUC usecases_port.RemoveFromFavoritesUseCasePort
	getObjectsUC    usecases_port.GetUserFavoritesUseCasePort
	getIdsUC 		usecases_port.GetUserFavoritesIdsUseCasePort
}

// NewFavoritesHandler - конструктор.
func NewFavoritesHandler(addUC usecases_port.AddToFavoritesUseCasePort, 
	removeUC usecases_port.RemoveFromFavoritesUseCasePort, 
	getObjectsUC usecases_port.GetUserFavoritesUseCasePort,
	getIdsUC usecases_port.GetUserFavoritesIdsUseCasePort) *FavoritesHandler {
	return &FavoritesHandler{
		addUC:    addUC,
		removeUC: removeUC,
		getObjectsUC:    getObjectsUC,
		getIdsUC: getIdsUC,
	}
}

func (h *FavoritesHandler) GetUserFavoritesIds(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context()).WithFields(port.Fields{"handler": "GetUserFavoritesIds"})
	
	// Извлекаем userID из контекста, который был добавлен middleware
	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		logger.Error("Invalid or missing user ID in context", nil, nil) 
		WriteJSONError(w, http.StatusUnauthorized, "Invalid user ID in context")
		return
	}

	handlerLogger := logger.WithFields(port.Fields{
		"user_id": userID,
	})
	handlerLogger.Info("Processing request to get user favorites ids", nil)

	ids, err := h.getIdsUC.Execute(r.Context(), userID)
	if err != nil {
		handlerLogger.Error("Get user favorites ids use case failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to retrieve favorites")
		return
	}

	RespondWithJSON(w, http.StatusOK, ids)
}

// GetUserFavorites обрабатывает GET /api/v1/favorites
func (h *FavoritesHandler) GetUserFavorites(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context()).WithFields(port.Fields{"handler": "GetUserFavorites"})
	
	// Извлекаем userID из контекста, который был добавлен middleware
	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		logger.Error("Invalid or missing user ID in context", nil, nil) 
		WriteJSONError(w, http.StatusUnauthorized, "Invalid user ID in context")
		return
	}
	
	// Парсим параметры пагинации
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 20 // Значение по умолчанию
	}
	if offset < 0 {
		offset = 0
	}

	handlerLogger := logger.WithFields(port.Fields{
		"user_id": userID,
		"limit":   limit,
		"offset":  offset,
	})
	handlerLogger.Info("Processing request to get user favorites", nil)
	
	// Вызываем Use Case
	paginatedResult, err := h.getObjectsUC.Execute(r.Context(), userID, limit, offset)
	if err != nil {
		handlerLogger.Error("Get user favorites use case failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to retrieve favorites")
		return
	}

	// Маппим результат из домена в DTO ответа
	response := PaginatedFavoritesResponse{
		Data:       make([]ObjectCardResponse, len(paginatedResult.Objects)),
		Total:      paginatedResult.TotalCount,
		Page:       paginatedResult.CurrentPage,
		PerPage:    paginatedResult.ItemsPerPage,
	}
	for i, obj := range paginatedResult.Objects {
		response.Data[i] = ObjectCardResponse{
			ID:       obj.ID,
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

	handlerLogger.Info("Successfully retrieved user favorites", port.Fields{
		"total_found": paginatedResult.TotalCount,
		"items_on_page": len(paginatedResult.Objects),
	})
	RespondWithJSON(w, http.StatusOK, response)
}

// AddToFavorites обрабатывает POST /api/v1/favorites
func (h *FavoritesHandler) AddToFavorites(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context()).WithFields(port.Fields{ "handler": "AddToFavorites"})

	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		logger.Error("Invalid or missing user ID in context", nil, nil)
		WriteJSONError(w, http.StatusUnauthorized, "Invalid user ID in context")
		return
	}

	var reqDTO AddFavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&reqDTO); err != nil {
		logger.Warn("Failed to decode request body for add favorite", port.Fields{"error": err.Error()})
		WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	masterObjectID, err := uuid.Parse(reqDTO.MasterObjectID)
	if err != nil {
		logger.Warn("Invalid master_object_id format in request", port.Fields{"provided_id": reqDTO.MasterObjectID})
		WriteJSONError(w, http.StatusBadRequest, "Invalid master_object_id format")
		return
	}

	handlerLogger := logger.WithFields(port.Fields{
		"user_id":          userID,
		"master_object_id": masterObjectID,
	})
	handlerLogger.Info("Processing request to add to favorites", nil)

	if err := h.addUC.Execute(r.Context(), userID, masterObjectID); err != nil {
		handlerLogger.Error("Add to favorites use case failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to add to favorites")
		return
	}

	handlerLogger.Info("Successfully added object to favorites", nil)
	w.WriteHeader(http.StatusCreated)
}

// RemoveFromFavorites обрабатывает DELETE /api/v1/favorites/{masterObjectID}
func (h *FavoritesHandler) RemoveFromFavorites(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context()).WithFields(port.Fields{"handler": "RemoveFromFavorites"})

	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		logger.Error("Invalid or missing user ID in context", nil, nil)
		WriteJSONError(w, http.StatusUnauthorized, "Invalid user ID in context")
		return
	}

	// Получаем ID из URL-параметра
	masterObjectIDStr := chi.URLParam(r, "masterObjectID")
	masterObjectID, err := uuid.Parse(masterObjectIDStr)
	if err != nil {
		logger.Warn("Invalid masterObjectID in URL", port.Fields{"provided_id": masterObjectIDStr})
		WriteJSONError(w, http.StatusBadRequest, "Invalid masterObjectID in URL")
		return
	}

	handlerLogger := logger.WithFields(port.Fields{
		"user_id":          userID,
		"master_object_id": masterObjectID,
	})
	handlerLogger.Info("Processing request to remove from favorites", nil)
	
	if err := h.removeUC.Execute(r.Context(), userID, masterObjectID); err != nil {
		handlerLogger.Error("Remove from favorites use case failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to remove from favorites")
		return
	}

	handlerLogger.Info("Successfully removed object from favorites", nil)
	w.WriteHeader(http.StatusNoContent) // 204 No Content - стандартный ответ на успешный DELETE
}