package logger_adapter

import (
	"actualization-service/internal/core/port"
	"fmt"
)

// MultiLoggerAdapter реализует LoggerPort, перенаправляя вызовы
// на несколько других логгеров.
type MultiLoggerAdapter struct {
	loggers []port.LoggerPort
}

// New создает новый композитный логгер.
func NewMultiloggerAdapter(loggers ...port.LoggerPort) (port.LoggerPort, error) {
	if len(loggers) == 0 {
		return nil, fmt.Errorf("multilogger: at least one logger is required")
	}
	return &MultiLoggerAdapter{loggers: loggers}, nil
}

func (m *MultiLoggerAdapter) Info(msg string, fields port.Fields) {
	for _, logger := range m.loggers {
		logger.Info(msg, fields)
	}
}

func (m *MultiLoggerAdapter) Warn(msg string, fields port.Fields) {
	for _, logger := range m.loggers {
		logger.Warn(msg, fields)
	}
}

func (m *MultiLoggerAdapter) Error(msg string, err error, fields port.Fields) {
	for _, logger := range m.loggers {
		logger.Error(msg, err, fields)
	}
}

// WithFields создает новый MultiLogger, где каждый из дочерних логгеров
// также был создан с помощью WithFields. Это сохраняет контекст для всех.
func (m *MultiLoggerAdapter) WithFields(fields port.Fields) port.LoggerPort {
	// Создаем новый срез для "обогащенных" логгеров
	enrichedLoggers := make([]port.LoggerPort, 0, len(m.loggers))

	for _, logger := range m.loggers {
		// Каждый дочерний логгер получает свой собственный обогащенный экземпляр
		enrichedLoggers = append(enrichedLoggers, logger.WithFields(fields))
	}

	// Возвращаем новый экземпляр MultiLogger с уже обогащенными дочерними логгерами
	return &MultiLoggerAdapter{loggers: enrichedLoggers}
}
