package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"kufar-parser-service/internal/contextkeys"
	"kufar-parser-service/internal/core/domain"
	"kufar-parser-service/internal/core/port"
	"real-estate-system/pkg/rabbitmq/rabbitmq_producer" // Путь к вашему пакету продюсера

	// "log"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const PARSE_NEW = 3

// RabbitMQLinkQueueAdapter реализует интерфейс PropertyLinkQueuePort для RabbitMQ.
type RabbitMQLinkQueueAdapter struct {
	producer   *rabbitmq_producer.Publisher
	routingKey string // Ключ маршрутизации для отправки ссылок
	// Можно добавить ExchangeName, если он не задан глобально в producer'е
	// exchangeName string
}

// NewRabbitMQLinkQueueAdapter создает новый экземпляр RabbitMQLinkQueueAdapter.
// producer - это уже инициализированный экземпляр вашего rabbitmq_producer.Publisher.
// routingKey - ключ, с которым будут публиковаться сообщения (например, "link.task.kufar").
func NewRabbitMQLinkQueueAdapter(producer *rabbitmq_producer.Publisher, routingKey string) (*RabbitMQLinkQueueAdapter, error) {
	if producer == nil {
		return nil, fmt.Errorf("rabbitmq adapter: producer cannot be nil")
	}
	if routingKey == "" {
		return nil, fmt.Errorf("rabbitmq adapter: routingKey cannot be empty")
	}

	return &RabbitMQLinkQueueAdapter{
		producer:   producer,
		routingKey: routingKey,
	}, nil
}

// Enqueue отправляет ссылку в очередь RabbitMQ.
func (a *RabbitMQLinkQueueAdapter) Enqueue(ctx context.Context, link domain.PropertyLink, taskID uuid.UUID) error {
	logger := contextkeys.LoggerFromContext(ctx)
	adapterLogger := logger.WithFields(port.Fields{
		"component":   "RabbitMQLinkQueueAdapter",
		"routing_key": a.routingKey,
	})

	linkTask := LinkTaskDTO{
		Source: link.Source,
		AdID:   link.AdID,
		TaskID: taskID,
	}

	linkJSON, err := json.Marshal(linkTask)
	if err != nil {
		adapterLogger.Error("Failed to marshal property link to JSON", err, port.Fields{"ad_id": link.AdID})
	}

	msg := amqp.Publishing{
		ContentType:  "application/json", // Указываем, что отправляем JSON
		Body:         linkJSON,
		DeliveryMode: amqp.Persistent, // Для сохранения сообщений при перезапуске брокера
		Timestamp:    time.Now(),
		Priority:     PARSE_NEW,
		Headers:      make(amqp.Table),
		// Можно добавить AppId или другие свойства, если необходимо
		// AppId: "parser-project",
	}

	// Пробрасываем trace_id в заголовки сообщения
	traceID := contextkeys.TraceIDFromContext(ctx)
	if traceID != "" {
		msg.Headers["x-trace-id"] = traceID
	}

	// Устанавливаем таймаут на операцию публикации, если контекст его не предоставляет
	publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Таймаут 10 секунд на публикацию
	defer cancel()

	err = a.producer.Publish(publishCtx, a.routingKey, msg)
	if err != nil {
		adapterLogger.Error("Failed to publish link", err, port.Fields{"ad_id": link.AdID})
		return fmt.Errorf("rabbitmq adapter: failed to publish link with AdID %d: %w", link.AdID, err)
	}

	adapterLogger.Debug("Successfully published link", port.Fields{"ad_id": link.AdID})
	return nil
}
