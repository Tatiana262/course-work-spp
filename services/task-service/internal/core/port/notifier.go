package port

import (
	"context"
	"task-service/internal/core/domain"
)

// TaskEvent - событие, которое мы отправляем подписчикам
type TaskEvent struct {
    Type string      `json:"type"` 
    Data domain.Task `json:"data"`
}

// NotifierPort - контракт для отправки уведомлений в реальном времени
type NotifierPort interface {
    Notify(ctx context.Context, event TaskEvent)
}