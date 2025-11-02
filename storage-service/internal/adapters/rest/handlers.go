package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"storage-service/internal/core/port"
)

type ActualiztionObjectsHandler struct {
    getActiveObjectsUC port.GetActiveObjectsUseCase
	getArchivedObjectsUC port.GetArchivedObjectsUseCase
	getObjectByIDUC port.GetObjectByIDUseCase
}



func NewPropertyHandlers(getActiveObjectsUC port.GetActiveObjectsUseCase, 
						getArchivedObjectsUC port.GetArchivedObjectsUseCase,
						getObjectByIDUC port.GetObjectByIDUseCase) *ActualiztionObjectsHandler {
    return &ActualiztionObjectsHandler{
		getActiveObjectsUC: getActiveObjectsUC,
		getArchivedObjectsUC: getArchivedObjectsUC,
		getObjectByIDUC: getObjectByIDUC,
	}
}


func (h *ActualiztionObjectsHandler) GetActiveObjects(w http.ResponseWriter, r *http.Request) {

	
	limit, err := GetLimitOrDefault(r)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "ActiveObjectsHandler: invalid limit value")
		return
	}

	offset, err := GetOffsetOrDefault(r)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "ActiveObjectsHandler: invalid offset value")
		return
	}

    properties, err := h.getActiveObjectsUC.FindActiveIDsForActualization(r.Context(), *limit, *offset)
    if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("ActiveObjectsHandler: failed to find IDs for actualization: %v", err))
        return
    }

    // Маппинг из доменной модели в DTO для ответа
    response := make([]PropertyInfoResponse, len(properties))
    for i, p := range properties {
        response[i] = PropertyInfoResponse{
            ID:        p.ID.String(),
            SourceAdID: p.SourceAdID,
            Source:    p.Source,
            UpdatedAt: p.UpdatedAt,
        }
    }

	// Отправить успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Errorf("ActiveObjectsHandler: failed to send response: %w", err)
	}

}


func (h *ActualiztionObjectsHandler) GetArchivedObjects(w http.ResponseWriter, r *http.Request) {

	
	limit, err := GetLimitOrDefault(r)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "ArchivedObjectsHandler: invalid limit value")
		return
	}

	offset, err := GetOffsetOrDefault(r)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "ArchivedObjectsHandler: invalid offset value")
		return
	}

    properties, err := h.getArchivedObjectsUC.FindArchivedIDsForActualization(r.Context(), *limit, *offset)
    if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("ArchivedObjectsHandler: failed to find IDs for actualization: %v", err))
        return
    }

    // Маппинг из доменной модели в DTO для ответа
    response := make([]PropertyInfoResponse, len(properties))
    for i, p := range properties {
        response[i] = PropertyInfoResponse{
            ID:        p.ID.String(),
            SourceAdID: p.SourceAdID,
            Source:    p.Source,
            UpdatedAt: p.UpdatedAt,
        }
    }

	// Отправить успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Errorf("ArchivedObjectsHandler: failed to send response: %w", err)
	}

}



func (h *ActualiztionObjectsHandler) GetObjectByID(w http.ResponseWriter, r *http.Request) {

	idStr := r.URL.Query().Get("id")
    if idStr == "" {
		WriteJSONError(w, http.StatusBadRequest, "GetObjectByID: id is required")
		return
	}
	

    property, err := h.getObjectByIDUC.FindObjectByIDForActualization(r.Context(), idStr)
    if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, fmt.Sprintf("ObjectByIDHandler: failed to find object for actualization: %v", err))
        return
    }

    
	// Отправить успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(property); err != nil {
		fmt.Errorf("ObjectByIDHandler: failed to send response: %w", err)
	}

}