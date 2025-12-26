package port

// Fields — это тип для передачи структурированных данных в лог.
type Fields map[string]interface{}

// LoggerPort определяет контракт для системы логирования.
// Он абстрагирует ядро приложения от конкретной реализации логгера.
type LoggerPort interface {
	// Info записывает информационное сообщение.
	Info(msg string, fields Fields)

	// Warn записывает предупреждение.
	Warn(msg string, fields Fields)

	// Error записывает ошибку, обычно вместе с объектом error.
	Error(msg string, err error, fields Fields)

	Debug(msg string, fields Fields)

	// WithFields создает новый экземпляр логгера с уже добавленными полями.
	// Это полезно для добавления контекста (например, request_id).
	WithFields(fields Fields) LoggerPort
}