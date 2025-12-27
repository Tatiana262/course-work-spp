package rest

import (
	"task-service/internal/core/domain"
	"time"
)

type CreateTaskRequest struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	CreatedByUserID string `json:"created_by_user_id"`
	ObjectID		string `json:"object_id,omitempty"`
}

type UpdateTaskRequest struct {
	Status        domain.TaskStatus `json:"status"`
	// ResultSummary *json.RawMessage  `json:"result_summary"`
}

// TaskResponse - DTO для ответа с одной задачей
type TaskResponse struct {
	ID              string               `json:"id"`
	Name            string               `json:"name"`
	Type            string               `json:"type"`
	Status          domain.TaskStatus    `json:"status"`
	ResultSummary   domain.ResultSummary `json:"result_summary"`
	CreatedAt       string               `json:"created_at"`
	StartedAt       *string              `json:"started_at,omitempty"`
	FinishedAt      *string              `json:"finished_at,omitempty"`
	CreatedByUserID string               `json:"created_by_user_id"`
}

// PaginatedTasksResponse - DTO для ответа со списком задач
type PaginatedTasksResponse struct {
	Data    []TaskResponse `json:"data"`
	Total   int64          `json:"total"`
	Page    int            `json:"page"`
	PerPage int            `json:"perPage"`
}


// toTaskResponse - маппер из доменной модели в DTO
func toTaskResponse(task *domain.Task) TaskResponse {
	resp := TaskResponse{
		ID:              task.ID.String(),
		Name:            task.Name,
		Type:            task.Type,
		Status:          task.Status,
		ResultSummary:   task.ResultSummary,
		CreatedAt:       task.CreatedAt.Format(time.RFC3339),
		CreatedByUserID: task.CreatedByUserID.String(),
	}
	if task.StartedAt != nil {
		startedAt := task.StartedAt.Format(time.RFC3339)
		resp.StartedAt = &startedAt
	}
	if task.FinishedAt != nil {
		finishedAt := task.FinishedAt.Format(time.RFC3339)
		resp.FinishedAt = &finishedAt
	}
	return resp
}