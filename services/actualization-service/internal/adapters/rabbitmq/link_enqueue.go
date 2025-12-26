package rabbitmq

import (
	"actualization-service/internal/constants"
	"actualization-service/internal/contextkeys"
	"actualization-service/internal/core/domain"
	"actualization-service/internal/core/port"
	"context"
	"encoding/json"
	"fmt"
	"real-estate-system/pkg/rabbitmq/rabbitmq_producer"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQLinkQueueAdapter реализует интерфейс LinksQueuePort для RabbitMQ
type RabbitMQLinkQueueAdapter struct {
	producer *rabbitmq_producer.Publisher
}

// NewRabbitMQLinkQueueAdapter создает новый экземпляр RabbitMQLinkQueueAdapter
// producer - это уже инициализированный экземпляр rabbitmq_producer.Publisher
// routingKey - ключ, с которым будут публиковаться сообщения
func NewRabbitMQLinkQueueAdapter(producer *rabbitmq_producer.Publisher) (*RabbitMQLinkQueueAdapter, error) {
	if producer == nil {
		return nil, fmt.Errorf("rabbitmq adapter: producer cannot be nil")
	}

	return &RabbitMQLinkQueueAdapter{
		producer: producer,
	}, nil
}

func (a *RabbitMQLinkQueueAdapter) PublishTask(ctx context.Context, task domain.ActualizationTask) error {

	logger := contextkeys.LoggerFromContext(ctx)
	adapterLogger := logger.WithFields(port.Fields{
		"component":   "RabbitMQLinkQueueAdapter",
		"source": 	   task.Source,
		"ad_url":      task.Task.Link,
		"task_id":     task.Task.TaskID.String(),
	})

	var routingKey string
	if task.Source == domain.KUFAR_SOURCE {
		routingKey = constants.RoutingKeyLinkTasksKufar
	}
	if task.Source == domain.REALT_SOURCE {
		routingKey = constants.RoutingKeyLinkTasksRealt
	}

	taskJSON, err := json.Marshal(task.Task)
	if err != nil {
		adapterLogger.Error("Failed to marshal task to JSON", err, nil)
		return fmt.Errorf("rabbitmq adapter: failed to marshal task to JSON for %s: %w", task.Task.Link, err)
	}

	msg := amqp.Publishing{
		ContentType:  "application/json", // Указываем, что отправляем JSON
		Body:         taskJSON,
		DeliveryMode: amqp.Persistent, // Для сохранения сообщений при перезапуске брокера
		Timestamp:    time.Now(),
		Priority:     task.Priority,
		Headers:      make(amqp.Table),
	}

	traceID := contextkeys.TraceIDFromContext(ctx)
	if traceID != "" {
		msg.Headers["x-trace-id"] = traceID
	}

	publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Таймаут 10 секунд на публикацию
	defer cancel()

	adapterLogger.Debug("Publishing actualization task", nil)
	err = a.producer.Publish(publishCtx, routingKey, msg)
	if err != nil {
		adapterLogger.Error("Failed to publish actualization task", err, nil)
		return fmt.Errorf("rabbitmq adapter: failed to publish task for %s: %w", task.Task.Link, err)
	}

	adapterLogger.Info("Successfully published actualization task", nil)
	return nil
}
