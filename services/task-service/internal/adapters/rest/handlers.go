package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	// "log"
	"net/http"
	"strconv"
	"task-service/internal/adapters/notifier"
	"task-service/internal/contextkeys"
	"task-service/internal/core/domain"
	"task-service/internal/core/port"
	"task-service/internal/core/port/usecases_port"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)


type TaskHandler struct {
	createTaskUC usecases_port.CreateTaskUseCasePort
	updateTaskUC usecases_port.UpdateTaskStatusUseCasePort
	getTaskUC    usecases_port.GetTaskByIdUseCasePort
	getTasksUC   usecases_port.GetTasksListUseCasePort
	processResultUC usecases_port.ProcessTaskResultUseCasePort
	notifier     *notifier.SSENotifier
}

// NewTaskHandler - конструктор
func NewTaskHandler(
	createUC usecases_port.CreateTaskUseCasePort,
	updateUC usecases_port.UpdateTaskStatusUseCasePort,
	getUC usecases_port.GetTaskByIdUseCasePort,
	getTasksUC usecases_port.GetTasksListUseCasePort,
	processResultUC usecases_port.ProcessTaskResultUseCasePort,
	notifier *notifier.SSENotifier,
) *TaskHandler {
	return &TaskHandler{
		createTaskUC: createUC,
		updateTaskUC: updateUC,
		getTaskUC:    getUC,
		getTasksUC:   getTasksUC,
		processResultUC: processResultUC,
		notifier:     notifier,
	}
}



func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context()).WithFields(port.Fields{"handler":  "CreateTask"})
	
	var req CreateTaskRequest
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Failed to decode create task request body", port.Fields{"error": err.Error()})
		WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.Type == "" || req.CreatedByUserID == "" {
		logger.Warn("Fields 'name', 'type', and 'created_by_user_id' are required", nil)
		WriteJSONError(w, http.StatusBadRequest, "Fields 'name', 'type', and 'created_by_user_id' are required")
		return
	}
	userID, err := uuid.Parse(req.CreatedByUserID)
	if err != nil {
		logger.Warn("Invalid 'created_by_user_id' format", port.Fields{"error": err.Error()})
		WriteJSONError(w, http.StatusBadRequest, "Invalid 'created_by_user_id' format")
		return
	}

	handlerLogger := logger.WithFields(port.Fields{
		"task_name": req.Name,
		"task_type": req.Type,
		"user_id":  userID.String(),
	})
	handlerLogger.Info("Processing request to create task", nil)

	var task *domain.Task
	if req.Type == "ACTUALIZE_BY_ID" {
		task, err = h.createTaskUC.Execute(r.Context(), req.Name, req.Type, userID, req.ObjectID)
	} else {
		task, err = h.createTaskUC.Execute(r.Context(), req.Name, req.Type, userID)
	}
	
	if err != nil {
		handlerLogger.Error("CreateTask use case failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to create task")
		return
	}

	handlerLogger.Info("Task created successfully", port.Fields{"task_id": task.ID.String()})
	RespondWithJSON(w, http.StatusCreated, toTaskResponse(task))
}


func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context()).WithFields(port.Fields{"handler":  "UpdateTask"})

	taskID, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		logger.Warn("Invalid task ID format in URL", port.Fields{"provided_id": chi.URLParam(r, "taskID")})
		WriteJSONError(w, http.StatusBadRequest, "Invalid task ID in URL")
		return
	}

	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Failed to decode update task request body", port.Fields{"error": err.Error()})
		WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// var summary domain.ResultSummary
	// if req.ResultSummary != nil {
	// 	if err := json.Unmarshal(*req.ResultSummary, &summary); err != nil {
	// 		logger.Warn("Invalid 'result_summary' format", port.Fields{"error": err.Error()})
	// 		WriteJSONError(w, http.StatusBadRequest, "Invalid 'result_summary' format")
	// 		return
	// 	}
	// }

	handlerLogger := logger.WithFields(port.Fields{
		"task_id":  taskID.String(),
		"status":   req.Status,
	})
	handlerLogger.Info("Processing request to update task", nil)

	task, err := h.updateTaskUC.Execute(r.Context(), taskID, req.Status)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			handlerLogger.Warn("Update failed: task not found", nil)
            WriteJSONError(w, http.StatusNotFound, err.Error())
            return
        }
		
		handlerLogger.Error("UpdateTask use case failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to update task")
		return
	}

	handlerLogger.Info("Task updated successfully", nil)
	RespondWithJSON(w, http.StatusOK, toTaskResponse(task))
}


func (h *TaskHandler) GetTasksList(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context()).WithFields(port.Fields{"handler": "GetTasksList"})

	// userID должен быть извлечен из контекста middleware аутентификации
	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		logger.Error("Invalid or missing user ID in context", nil, nil)
		WriteJSONError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	query := r.URL.Query()

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

	handlerLogger := logger.WithFields(port.Fields{
		"user_id": userID,
		"limit":   limit,
		"offset":  offset,
	})
	handlerLogger.Info("Processing request to get tasks list", nil)

	tasks, totalCount, err := h.getTasksUC.Execute(r.Context(), userID, limit, offset)
	if err != nil {
		handlerLogger.Error("GetTasksList use case failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to retrieve tasks")
		return
	}

	handlerLogger.Info("Successfully retrieved tasks list", port.Fields{
		"total_found": totalCount,
		"items_on_page": len(tasks),
	})

	// Маппинг в DTO
	taskResponses := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		taskResponses[i] = toTaskResponse(&task)
	}

	paginatedResponse := PaginatedTasksResponse{
		Data:    taskResponses,
		Total:   totalCount,
		Page:    offset/limit + 1,
		PerPage: limit,
	}

	RespondWithJSON(w, http.StatusOK, paginatedResponse)
}


func (h *TaskHandler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context()).WithFields(port.Fields{"handler": "GetTaskByID"})

	taskID, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		logger.Warn("Invalid task ID format in URL", port.Fields{"provided_id": chi.URLParam(r, "taskID")})
		WriteJSONError(w, http.StatusBadRequest, "Invalid task ID in URL")
		return
	}
	

	handlerLogger := logger.WithFields(port.Fields{
		"task_id": taskID.String(),
	})
	handlerLogger.Info("Processing request to get task by ID", nil)

	task, err := h.getTaskUC.Execute(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			handlerLogger.Warn("Get task failed: task not found", nil)
            WriteJSONError(w, http.StatusNotFound, err.Error())
            return
        }
		handlerLogger.Error("GetTaskByID use case failed", err, nil)
		WriteJSONError(w, http.StatusInternalServerError, "Failed to retrieve task")
		return
	}

	handlerLogger.Info("Successfully retrieved task by ID", nil)
	RespondWithJSON(w, http.StatusOK, toTaskResponse(task))
}


// SubscribeToTasks - обработчик для GET /api/v1/tasks/subscribe
func (h *TaskHandler) SubscribeToTasks(w http.ResponseWriter, r *http.Request) {
	logger := contextkeys.LoggerFromContext(r.Context()).WithFields(port.Fields{"handler": "SubscribeToTasks"})

	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		logger.Error("User ID in context for SSE subscription invalid or missing", nil, nil)
		WriteJSONError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	handlerLogger := logger.WithFields(port.Fields{
		"user_id": userID,
	})
	handlerLogger.Info("New client subscribing to SSE events", nil)


	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	
	clientChan := h.notifier.AddClient(userID.String())
	defer h.notifier.RemoveClient(userID.String(), clientChan)

	// Отправляем ping для подтверждения установки соединения
	fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	if f, ok := w.(http.Flusher); ok { f.Flush() }

	// Отправляем пустой комментарий каждые 15 секунд
	ticker := time.NewTicker(15 * time.Second)
    defer ticker.Stop()

	for {
		select {
		case data := <-clientChan:
			if _, err := fmt.Fprintf(w, "%s", data); err != nil {
				handlerLogger.Error("Error writing to client, closing SSE connection", err, nil)
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			handlerLogger.Info("Sent SSE event to client", nil)

		case <-ticker.C:
            // В спецификации SSE строки, начинающиеся с двоеточия (:), считаются комментариями
            // Браузер их получает, канал остается активным, но JS-код (onmessage) их игнорирует
            if _, err := fmt.Fprintf(w, ": keep-alive\n\n"); err != nil {
                return
            }
            if f, ok := w.(http.Flusher); ok { f.Flush() }
            // handlerLogger.Debug("Sent keep-alive ping", nil) //  для отладки

		case <-r.Context().Done():
			handlerLogger.Info("SSE client disconnected.", nil)
			return
		}
	}
}