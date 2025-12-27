package domain

import (
	"time"

	"github.com/google/uuid"
)

// TaskStatus - перечисление для статусов задачи
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusRunning   TaskStatus = "running"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
)

// ResultSummary - структура для хранения сводной информации о результатах
// Использование map[string]interface{} делает ее гибкой для разных типов задач
type ResultSummary map[string]interface{}

// Task - основная доменная сущность
type Task struct {
	ID               uuid.UUID		`json:"id"`
	Name             string			`json:"name"`
	Type             string			`json:"type"`
	Status           TaskStatus		`json:"status"`
	ResultSummary    ResultSummary	`json:"result_summary"`
	CreatedAt        time.Time		`json:"created_at"`
	StartedAt        *time.Time 	`json:"started_at"`
	FinishedAt       *time.Time 	`json:"finished_at"`
	CreatedByUserID  uuid.UUID		`json:"created_by_user_id"`
	TargetObjectID 	 *uuid.UUID 	`json:"target_object_id,omitempty"` 
}

// NewTask - конструктор для создания новой задачи
func NewTask(name, taskType string, createdByUserID uuid.UUID) *Task {
	return &Task{
		ID:              uuid.New(),
		Name:            name,
		Type:            taskType,
		Status:          StatusPending, // Начальный статус
		CreatedAt:       time.Now().UTC(),
		CreatedByUserID: createdByUserID,
		ResultSummary: make(ResultSummary),
	}
}