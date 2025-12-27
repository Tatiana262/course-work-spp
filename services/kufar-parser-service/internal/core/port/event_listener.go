package port

import "context"

// EventListenerPort определяет контракт для компонента, который слушает
// внешние события (сообщения из очереди) и запускает
// соответствующую бизнес-логику
type EventListenerPort interface {
	// Start запускает слушателя
	Start(ctx context.Context) error

	// Close корректно останавливает слушателя, дожидаясь завершения активных задач
	Close() error
}