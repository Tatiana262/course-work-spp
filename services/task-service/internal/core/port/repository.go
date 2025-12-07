package port

import (
    "context"
    "task-service/internal/core/domain" // Убедитесь, что путь верный
    "github.com/google/uuid"
)

type TaskRepositoryPort interface {
    Create(ctx context.Context, task *domain.Task) error
    Update(ctx context.Context, task *domain.Task) error
    FindByID(ctx context.Context, taskID uuid.UUID) (*domain.Task, error)
    FindAll(ctx context.Context, createdByUserID uuid.UUID, limit, offset int) ([]domain.Task, int64, error)
    // Метод для инкрементального обновления результатов
    IncrementSummary(ctx context.Context, taskID uuid.UUID, results map[string]int) error
}