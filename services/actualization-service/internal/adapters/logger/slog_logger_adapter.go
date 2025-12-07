package logger_adapter

import (
	// "context"
	"actualization-service/internal/core/port"
	"io"
	"log/slog"
	"os"
	"github.com/lmittmann/tint" 
)

// SlogAdapter реализует LoggerPort с использованием стандартной библиотеки slog.
type SlogAdapter struct {
	logger *slog.Logger
}

// Config для SlogAdapter
type SlogConfig struct {
	// Writer - куда писать логи. По умолчанию os.Stdout.
	Writer io.Writer
	// Level - уровень логирования (slog.LevelInfo, slog.LevelDebug, etc.).
	Level slog.Leveler
	// AddSource - добавлять ли в лог информацию о файле и строке кода.
	AddSource bool
	// IsJSON - использовать ли JSON формат. По умолчанию - текстовый.
	IsJSON bool
	UseColor  bool
}

// NewSlogAdapter создает новый экземпляр адаптера.
func NewSlogAdapter(cfg SlogConfig) port.LoggerPort {
	if cfg.Writer == nil {
		cfg.Writer = os.Stdout
	}
	if cfg.Level == nil {
		cfg.Level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		AddSource: cfg.AddSource,
		Level:     cfg.Level,
	}

	var handler slog.Handler
	if cfg.IsJSON {
		handler = slog.NewJSONHandler(cfg.Writer, opts)
	} else if cfg.UseColor {
		// Если нужны цвета, используем tint.NewHandler
		tintOpts := &tint.Options{
			Level:     cfg.Level,
			AddSource: cfg.AddSource,
			TimeFormat: "2006-01-02 15:04:05", // Более короткий и удобный формат времени
		}
		// tint автоматически определяет, поддерживает ли терминал цвета!
		handler = tint.NewHandler(cfg.Writer, tintOpts)
	} else {
		handler = slog.NewTextHandler(cfg.Writer, opts)
	}

	logger := slog.New(handler)
	return &SlogAdapter{logger: logger}
}

// fieldsToSlogAttrs конвертирует наш port.Fields в []slog.Attr
func (a *SlogAdapter) fieldsToSlogAttrs(fields port.Fields) []any {
	var attrs []any
	for k, v := range fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	return attrs
}

func (a *SlogAdapter) Info(msg string, fields port.Fields) {
	attrs := a.fieldsToSlogAttrs(fields)
	a.logger.Info(msg, attrs...)
}

func (a *SlogAdapter) Warn(msg string, fields port.Fields) {
	attrs := a.fieldsToSlogAttrs(fields)
	a.logger.Warn(msg, attrs...)
}

func (a *SlogAdapter) Error(msg string, err error, fields port.Fields) {
	attrs := a.fieldsToSlogAttrs(fields)
	if err != nil {
		// slog имеет специальную поддержку для поля "error"
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	a.logger.Error(msg, attrs...)
}

func (a *SlogAdapter) Debug(msg string, fields port.Fields) {
    attrs := a.fieldsToSlogAttrs(fields)
    a.logger.Debug(msg, attrs...) // slog уже умеет это делать
}

func (a *SlogAdapter) WithFields(fields port.Fields) port.LoggerPort {
	attrs := a.fieldsToSlogAttrs(fields)
	// slog.With создает новый логгер с добавленными атрибутами
	newLogger := a.logger.With(attrs...)
	return &SlogAdapter{logger: newLogger}
}