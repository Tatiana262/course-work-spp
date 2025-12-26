package fluentlogger

import (
	"fmt"
	"github.com/fluent/fluent-logger-golang/fluent"
)

// Config хранит конфигурацию для подключения к Fluent Bit
type Config struct {
	Host      string 
	Port      int    // 24224
	TagPrefix string // Общий префикс для всех тегов логов этого сервиса
}

// NewClient создает и возвращает новый клиент для Fluent Bit
func NewClient(cfg Config) (*fluent.Fluent, error) {
	if cfg.TagPrefix == "" {
		return nil, fmt.Errorf("fluentd tag prefix is required")
	}

	logger, err := fluent.New(fluent.Config{
		FluentHost: cfg.Host,
		FluentPort: cfg.Port,
		TagPrefix:  cfg.TagPrefix,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create fluentd logger: %w", err)
	}

	return logger, nil
}