package rest

import (
	"actualization-service/internal/contextkeys"
	"actualization-service/internal/core/port"
	"actualization-service/internal/core/port/usecases_port"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type ActualizationHandlers struct {
	actualizeActiveUC     usecases_port.ActualizeActiveObjectsUseCase
	actualizeArchivedUC   usecases_port.ActualizeArchivedObjectsUseCase
	actualizeObjectByIdUC usecases_port.ActualizeObjectByIdUseCase
	findNewObjectsUC      usecases_port.FindNewObjectsUseCase
}

// NewActualizationHandlers - конструктор для обработчиков
func NewActualizationHandlers(actualizeActiveUC usecases_port.ActualizeActiveObjectsUseCase,
	actualizeArchivedUC usecases_port.ActualizeArchivedObjectsUseCase,
	actualizeObjectByIdUC usecases_port.ActualizeObjectByIdUseCase,
	findNewObjectsUC usecases_port.FindNewObjectsUseCase) *ActualizationHandlers {
	return &ActualizationHandlers{
		actualizeActiveUC:     actualizeActiveUC,
		actualizeArchivedUC:   actualizeArchivedUC,
		actualizeObjectByIdUC: actualizeObjectByIdUC,
		findNewObjectsUC:      findNewObjectsUC,
	}
}

// HandleActualizeActive - обработчик для POST /api/v1/actualize/active
func (h *ActualizationHandlers) HandleActualizeActiveObjects(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context()).WithFields(port.Fields{"handler": "HandleActualizeActiveObjects"})

	userID, _ := r.Context().Value(userIDKey).(uuid.UUID)
	// Декодируем тело запроса в нашу DTO структуру
	var reqDTO ActualizeRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&reqDTO); err != nil {
		if err == io.EOF { // Если тело запроса пустое
			logger.Error("Failed to decode request body", err, nil)
			WriteJSONError(w, http.StatusBadRequest, "Request body is empty")
			return
		}
		WriteJSONError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if reqDTO.LimitPerCategory <= 0 {
		WriteJSONError(w, http.StatusBadRequest, "Field 'limit' must be a positive number")
		return
	}

	loggerForActualize := logger.WithFields(port.Fields{
		"limit":    reqDTO.LimitPerCategory,
		"category": *reqDTO.Category,
	})

	loggerForActualize.Info("Received request to actualize active objects for category", nil)

	// Вызываем Use Case
	taskID, err := h.actualizeActiveUC.Execute(r.Context(), userID, reqDTO.Category, reqDTO.LimitPerCategory)
	if err != nil {
		loggerForActualize.Error("Use case execution failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to start actualization process")
		return
	}

	loggerForActualize.Info("Successfully started actualization task", port.Fields{"task_id": taskID.String()})
	RespondWithJSON(w, http.StatusAccepted, map[string]string{"task_id": taskID.String()})
}

func (h *ActualizationHandlers) HandleActualizeArchivedObjects(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context()).WithFields(port.Fields{"handler": "HandleActualizeArchivedObjects"})

	userID, _ := r.Context().Value(userIDKey).(uuid.UUID)
	// Декодируем тело запроса в нашу DTO структуру
	var reqDTO ActualizeRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&reqDTO); err != nil {
		if err == io.EOF { // Если тело запроса пустое
			logger.Error("Failed to decode request body", err, nil)
			WriteJSONError(w, http.StatusBadRequest, "Request body is empty")
			return
		}
		WriteJSONError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if reqDTO.LimitPerCategory <= 0 {
		WriteJSONError(w, http.StatusBadRequest, "Field 'limit' must be a positive number")
		return
	}

	loggerForActualize := logger.WithFields(port.Fields{
		"limit":    reqDTO.LimitPerCategory,
		"category": *reqDTO.Category,
	})

	loggerForActualize.Info("Received request to actualize archived objects for category", nil)

	// Вызываем Use Case
	taskID, err := h.actualizeArchivedUC.Execute(r.Context(), userID, reqDTO.Category, reqDTO.LimitPerCategory)
	if err != nil {
		loggerForActualize.Error("Use case execution failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to start actualization process")
		return
	}

	// Отправляем успешный ответ
	// 202 Accepted - запрос принят
	loggerForActualize.Info("Successfully started actualization task", port.Fields{"task_id": taskID.String()})
	RespondWithJSON(w, http.StatusAccepted, map[string]string{"task_id": taskID.String()})
}

func (h *ActualizationHandlers) HandleActualizeObjectByID(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context()).WithFields(port.Fields{"handler": "HandleActualizeObjectByID"})

	userID, _ := r.Context().Value(userIDKey).(uuid.UUID)
	// Декодируем тело запроса в нашу DTO структуру
	var reqDTO ActualizeObjectDTO
	if err := json.NewDecoder(r.Body).Decode(&reqDTO); err != nil {
		if err == io.EOF { // Если тело запроса пустое
			logger.Error("Failed to decode request body", err, nil)
			WriteJSONError(w, http.StatusBadRequest, "Request body is empty")
			return
		}
		WriteJSONError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Валидируем входные данные
	if reqDTO.Id == "" {
		WriteJSONError(w, http.StatusBadRequest, "Field 'id' is required")
		return
	}

	loggerForActualize := logger.WithFields(port.Fields{"id": reqDTO.Id})
	loggerForActualize.Info("Received request to actualize object by id", nil)

	// Вызываем Use Case
	taskID, err := h.actualizeObjectByIdUC.Execute(r.Context(), userID, reqDTO.Id) // Приоритет 3 для активных
	if err != nil {
		loggerForActualize.Error("Use case execution failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to start actualization process")
		return
	}

	// Отправляем успешный ответ
	// 202 Accepted - запрос принят
	loggerForActualize.Info("Successfully started actualization task", port.Fields{"task_id": taskID.String()})
	RespondWithJSON(w, http.StatusAccepted, map[string]string{"task_id": taskID.String()})
}

func (h *ActualizationHandlers) HandleFindNewObjects(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context()).WithFields(port.Fields{"handler": "HandleFindNewObjects"})

	userID, _ := r.Context().Value(userIDKey).(uuid.UUID)
	// Декодируем тело запроса в нашу DTO структуру
	var reqDTO FindNewRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&reqDTO); err != nil {
		if err == io.EOF { // Если тело запроса пустое
			logger.Error("Failed to decode request body", err, nil)
			WriteJSONError(w, http.StatusBadRequest, "Request body is empty")
			return
		}
		WriteJSONError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	loggerForActualize := logger.WithFields(port.Fields{
		"categories": strings.Join(reqDTO.Categories, ", "),
		"regions":    strings.Join(reqDTO.Regions, ", "),
	})
	loggerForActualize.Info("Received request to parse new objects for categories in regions", nil)

	// Вызываем Use Case
	taskID, err := h.findNewObjectsUC.Execute(r.Context(), userID, reqDTO.Categories, reqDTO.Regions)
	if err != nil {
		loggerForActualize.Error("Use case execution failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to start parsing new objects process")
		return
	}

	// Отправляем успешный ответ
	// 202 Accepted - запрос принят
	loggerForActualize.Info("Successfully started actualization task", port.Fields{"task_id": taskID.String()})
	RespondWithJSON(w, http.StatusAccepted, map[string]string{"task_id": taskID.String()})
}
