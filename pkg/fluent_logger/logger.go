package fluentlogger

import (
	"fmt"
	"github.com/fluent/fluent-logger-golang/fluent"
)

// Config хранит конфигурацию для подключения к Fluent Bit.
type Config struct {
	Host      string // Например, "127.0.0.1" или "fluent-bit" в Docker
	Port      int    // Например, 24224
	TagPrefix string // Общий префикс для всех тегов логов этого сервиса
}

// NewClient создает и возвращает новый клиент для Fluent Bit.
func NewClient(cfg Config) (*fluent.Fluent, error) {
	if cfg.TagPrefix == "" {
		return nil, fmt.Errorf("fluentd tag prefix is required")
	}

	logger, err := fluent.New(fluent.Config{
		FluentHost: cfg.Host,
		FluentPort: cfg.Port,
		TagPrefix:  cfg.TagPrefix,
		// Можно добавить другие настройки, например, таймауты
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create fluentd logger: %w", err)
	}

	// Важно! Пинга как такового нет. Успешное создание клиента не гарантирует
	// соединение. Ошибки будут возникать при первой попытке отправки лога.
	// В реальном приложении можно добавить retry-логику или health-check.

	return logger, nil
}