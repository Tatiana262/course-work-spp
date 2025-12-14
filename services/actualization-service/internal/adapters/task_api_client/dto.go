package task_api_client

// DTO для создания задачи
type createTaskRequest struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	CreatedByUserID string `json:"created_by_user_id"`
	ObjectID		string `json:"object_id,omitempty"`
}

type createTaskResponse struct {
	ID string `json:"id"`
}

// DTO для обновления статуса
type updateTaskRequest struct {
	Status string `json:"status"`
}