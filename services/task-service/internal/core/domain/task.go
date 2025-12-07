package domain

import (
	"time"

	"github.com/google/uuid"
)

// TaskStatus - перечисление для статусов задачи.
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusRunning   TaskStatus = "running"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
)

// ResultSummary - структура для хранения сводной информации о результатах.
// Использование `map[string]interface{}` делает ее гибкой для разных типов задач.
type ResultSummary map[string]interface{}

// Task - основная доменная сущность.
type Task struct {
	ID               uuid.UUID
	Name             string
	Type             string
	Status           TaskStatus
	ResultSummary    ResultSummary
	CreatedAt        time.Time
	StartedAt        *time.Time // Указатель, т.к. может быть nil
	FinishedAt       *time.Time // Указатель, т.к. может быть nil
	CreatedByUserID  uuid.UUID
}

// NewTask - конструктор для создания новой задачи.
func NewTask(name, taskType string, createdByUserID uuid.UUID) *Task {
	return &Task{
		ID:              uuid.New(),
		Name:            name,
		Type:            taskType,
		Status:          StatusPending, // Начальный статус
		CreatedAt:       time.Now().UTC(),
		CreatedByUserID: createdByUserID,
	}
}