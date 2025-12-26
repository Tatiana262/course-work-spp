package contextkeys

import (
	"context"
	"authentication-service/internal/core/port"
)

// Тип для ключа контекста. Используем приватный тип, чтобы избежать коллизий.
type loggerKeyType struct{}

var loggerKey = loggerKeyType{}

// ContextWithLogger помещает логгер в контекст.
func ContextWithLogger(ctx context.Context, logger port.LoggerPort) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// LoggerFromContext извлекает логгер из контекста.
// Если логгер не найден, возвращает no-op логгер, чтобы избежать nil pointer panic.
func LoggerFromContext(ctx context.Context) port.LoggerPort {
	if logger, ok := ctx.Value(loggerKey).(port.LoggerPort); ok {
		return logger
	}
	// Возвращаем "пустышку", если в контексте нет логгера.
	// В реальном приложении здесь можно возвращать глобальный логгер или паниковать.
	return &noopLogger{}
}

// noopLogger - это реализация LoggerPort, которая ничего не делает.
type noopLogger struct{}
func (n *noopLogger) Info(msg string, fields port.Fields)                 {}
func (n *noopLogger) Warn(msg string, fields port.Fields)                 {}
func (n *noopLogger) Error(msg string, err error, fields port.Fields)     {}
func (n *noopLogger) Debug(msg string, fields port.Fields)				  {}
func (n *noopLogger) WithFields(fields port.Fields) port.LoggerPort       { return n }