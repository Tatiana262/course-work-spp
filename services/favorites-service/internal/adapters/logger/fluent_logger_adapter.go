package logger_adapter

import (
	"favorites-service/internal/core/port"
	"fmt"
	"log/slog"
	"time"

	"github.com/fluent/fluent-logger-golang/fluent"
)

// FluentLoggerAdapter реализует LoggerPort для отправки логов в Fluent Bit.
type FluentLoggerAdapter struct {
	client *fluent.Fluent
	fields port.Fields // Поля, добавленные через WithFields
	minLevel  slog.Level
}

// NewFluentLoggerAdapter создает новый экземпляр адаптера.
func NewFluentLoggerAdapter(client *fluent.Fluent, minLevel slog.Leveler) (*FluentLoggerAdapter, error) {
	if client == nil {
		return nil, fmt.Errorf("fluent client cannot be nil")
	}

	level := slog.LevelInfo
	if minLevel != nil {
		level = minLevel.Level()
	}

	return &FluentLoggerAdapter{
		client: client,
		fields: make(port.Fields),
		minLevel: level,
	}, nil
}

// mergeFields объединяет поля логгера с полями, переданными в вызов.
func (a *FluentLoggerAdapter) mergeFields(fields port.Fields) port.Fields {
	merged := make(port.Fields, len(a.fields)+len(fields))
	for k, v := range a.fields {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}
	return merged
}

// post отправляет данные в Fluent Bit.
func (a *FluentLoggerAdapter) post(level string, msg string, data port.Fields) {
	// Добавляем обязательные поля
	data["level"] = level
	data["message"] = msg
	data["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)

	// Тег в fluentd обычно используется для маршрутизации.
	// Например, 'app.info', 'app.error'.
	tag := level

	// Игнорируем ошибку, чтобы логирование не уронило приложение.
	// В продакшене можно добавить fallback-логгер в stdout.
	_ = a.client.Post(tag, data)
}

func (a *FluentLoggerAdapter) Info(msg string, fields port.Fields) {
	if a.minLevel > slog.LevelInfo { return }
	data := a.mergeFields(fields)
	a.post("info", msg, data)
}

func (a *FluentLoggerAdapter) Warn(msg string, fields port.Fields) {
	if a.minLevel > slog.LevelWarn { return }
	data := a.mergeFields(fields)
	a.post("warn", msg, data)
}

func (a *FluentLoggerAdapter) Error(msg string, err error, fields port.Fields) {
	if a.minLevel > slog.LevelError { return }
	data := a.mergeFields(fields)
	if err != nil {
		data["error"] = err.Error() // Добавляем текст ошибки в поля
	}
	a.post("error", msg, data)
}

func (a *FluentLoggerAdapter) Debug(msg string, fields port.Fields) {
    if a.minLevel > slog.LevelDebug { return } // Проверка уровня
    data := a.mergeFields(fields)
    a.post("debug", msg, data) // Отправляем с тегом "debug"
}

// WithFields создает новый логгер с расширенным контекстом.
func (a *FluentLoggerAdapter) WithFields(fields port.Fields) port.LoggerPort {
	// Создаем новый экземпляр адаптера, чтобы не изменять текущий (иммутабельность)
	return &FluentLoggerAdapter{
		client: a.client,
		fields: a.mergeFields(fields),
		minLevel: a.minLevel, 
	}
}

// Close закрывает соединение с Fluentd.
func (a *FluentLoggerAdapter) Close() error {
    return a.client.Close()
}