package port

// Fields — это тип для передачи структурированных данных в лог
type Fields map[string]interface{}

// LoggerPort определяет контракт для системы логирования
type LoggerPort interface {
	
	Info(msg string, fields Fields)

	Warn(msg string, fields Fields)

	Error(msg string, err error, fields Fields)

	Debug(msg string, fields Fields)
	// WithFields создает новый экземпляр логгера с уже добавленными полями
	WithFields(fields Fields) LoggerPort
}