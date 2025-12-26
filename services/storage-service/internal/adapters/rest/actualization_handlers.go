package rest

import (
	"fmt"
	"net/http"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/port"
	usecases_port "storage-service/internal/core/port/usecases_port"
)

type ActualiztionObjectsHandler struct {
    getActiveObjectsUC usecases_port.GetActiveObjectsUseCase
	getArchivedObjectsUC usecases_port.GetArchivedObjectsUseCase
	getObjectByIDUC usecases_port.GetObjectByIDUseCase
	getActualizationStatsUC usecases_port.GetActualizationStats
}



func NewActualizationHandlers(getActiveObjectsUC usecases_port.GetActiveObjectsUseCase, 
						getArchivedObjectsUC usecases_port.GetArchivedObjectsUseCase,
						getObjectByIDUC usecases_port.GetObjectByIDUseCase,
						getActualizationStatsUC usecases_port.GetActualizationStats) *ActualiztionObjectsHandler {
    return &ActualiztionObjectsHandler{
		getActiveObjectsUC: getActiveObjectsUC,
		getArchivedObjectsUC: getArchivedObjectsUC,
		getObjectByIDUC: getObjectByIDUC,
		getActualizationStatsUC: getActualizationStatsUC,
	}
}


func (h *ActualiztionObjectsHandler) GetActiveObjects(w http.ResponseWriter, r *http.Request) {

	logger := contextkeys.LoggerFromContext(r.Context())
	
	limit, err := GetLimitOrDefault(r)
	if err != nil {
		logger.Warn("Invalid 'limit' parameter", port.Fields{"error": err.Error()})
		WriteJSONError(w, http.StatusBadRequest, "ActiveObjectsHandler: invalid limit value")
		return
	}

	category := r.URL.Query().Get("category")
	if category == "" {
		logger.Warn("Missing 'category' parameter", nil)
		WriteJSONError(w, http.StatusBadRequest, "ActiveObjectsHandler: empty category value")
		return
	}

	handlerLogger := logger.WithFields(port.Fields{
		"handler": "GetActiveObjects",
		"limit":   *limit,
		"category": category,
	})
	handlerLogger.Info("Processing request", nil)

    properties, err := h.getActiveObjectsUC.FindActiveIDsForActualization(r.Context(), category, *limit)
    if err != nil {
		handlerLogger.Error("Use case failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("ActiveObjectsHandler: failed to find IDs for actualization: %v", err))
        return
    }

    // Маппинг из доменной модели в DTO для ответа
    response := make([]PropertyInfoResponse, len(properties))
    for i, p := range properties {
        response[i] = PropertyInfoResponse{
            // ID:        p.ID.String(),
			AdID:      p.AdID,
            AdLink: 	   p.Link,
            Source:    p.Source,
            // UpdatedAt: p.UpdatedAt,
        }
    }

	handlerLogger.Info("Successfully found objects", port.Fields{"count": len(response)})
	RespondWithJSON(w, http.StatusOK, response) // Используем хелпер для отправки
}


func (h *ActualiztionObjectsHandler) GetArchivedObjects(w http.ResponseWriter, r *http.Request) {

	logger := contextkeys.LoggerFromContext(r.Context())

	limit, err := GetLimitOrDefault(r)
	if err != nil {
		logger.Warn("Invalid 'limit' parameter", port.Fields{"error": err.Error()})
		WriteJSONError(w, http.StatusBadRequest, "ArchivedObjectsHandler: invalid limit value")
		return
	}
	category := r.URL.Query().Get("category")
	if category == "" {
		logger.Warn("Missing 'category' parameter", nil)
		WriteJSONError(w, http.StatusBadRequest, "ArchivedObjectsHandler: empty category value")
		return
	}

	handlerLogger := logger.WithFields(port.Fields{
		"handler": "GetArchivedObjects",
		"limit":   *limit,
		"category": category,
	})
	handlerLogger.Info("Processing request", nil)

    properties, err := h.getArchivedObjectsUC.FindArchivedIDsForActualization(r.Context(), category, *limit)
    if err != nil {
		handlerLogger.Error("Use case failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("ArchivedObjectsHandler: failed to find IDs for actualization: %v", err))
        return
    }

    // Маппинг из доменной модели в DTO для ответа
    response := make([]PropertyInfoResponse, len(properties))
    for i, p := range properties {
        response[i] = PropertyInfoResponse{
            // ID:        p.ID.String(),
            AdID:      p.AdID,
            AdLink: 	   p.Link,
            Source:    p.Source,
            // UpdatedAt: p.UpdatedAt,
        }
    }

	handlerLogger.Info("Successfully found objects", port.Fields{"count": len(response)})
	RespondWithJSON(w, http.StatusOK, response) // Используем хелпер для отправки
}



func (h *ActualiztionObjectsHandler) GetObjectsByMasterID(w http.ResponseWriter, r *http.Request) {

	logger := contextkeys.LoggerFromContext(r.Context())

	idStr := r.URL.Query().Get("id")
    if idStr == "" {
		logger.Warn("Missing 'id' parameter", nil)
		WriteJSONError(w, http.StatusBadRequest, "GetObjectByID: id is required")
		return
	}
	
	
	handlerLogger := logger.WithFields(port.Fields{
		"handler": "GetObjectByID",
		"id":   idStr,
	})
	handlerLogger.Info("Processing request", nil)

    properties, err := h.getObjectByIDUC.FindObjectsByIDForActualization(r.Context(), idStr)
    if err != nil {
		handlerLogger.Error("Use case failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("GetObjectsByMasterIDHandler: failed to find objects for actualization: %v", err))
        return
    }


	response := make([]PropertyInfoResponse, len(properties))
    for i, p := range properties {
        response[i] = PropertyInfoResponse{
            // ID:        p.ID.String(),
            AdID:      p.AdID,
            AdLink: 	   p.Link,
            Source:    p.Source,
            // UpdatedAt: p.UpdatedAt,
        }
    }

	handlerLogger.Info("Successfully found objects", port.Fields{"count": len(response)})
	RespondWithJSON(w, http.StatusOK, response) // Используем хелпер для отправки
}


func(h *ActualiztionObjectsHandler) GetActualizationStats(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context())

	handlerLogger := logger.WithFields(port.Fields{
		"handler": "GetActualizationStats",
	})

	stats, err := h.getActualizationStatsUC.Execute(r.Context())
    if err != nil {
		handlerLogger.Error("Use case failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("GetActualizationStatsHandler: failed to get actualization stats: %v", err))
        return
    }

	statsResp := make([]StatsByCategoryResponse, len(stats))
	for i, stat := range(stats) {
		statsResp[i] = StatsByCategoryResponse{
			DisplayName: stat.DisplayName,
			SystemName: stat.SystemName,
			ActiveCount: stat.ActiveCount,
			ArchivedCount: stat.ArchivedCount,
		}
	}

	
	handlerLogger.Info("Successfully get stats", nil)
	RespondWithJSON(w, http.StatusOK, statsResp) // Используем хелпер для отправки
}