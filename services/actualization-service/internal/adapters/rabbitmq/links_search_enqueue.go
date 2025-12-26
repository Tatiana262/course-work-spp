package rabbitmq

import (
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

// RabbitMQLinkQueueAdapter реализует интерфейс LinksSearchQueuePort для RabbitMQ
type RabbitMQLinksSearchQueueAdapter struct {
	producer *rabbitmq_producer.Publisher

}

func NewRabbitMQLinksSearchQueueAdapter(producer *rabbitmq_producer.Publisher) (*RabbitMQLinksSearchQueueAdapter, error) {
	if producer == nil {
		return nil, fmt.Errorf("rabbitmq adapter: producer cannot be nil")
	}

	return &RabbitMQLinksSearchQueueAdapter{
		producer: producer,
	}, nil
}

func (a *RabbitMQLinksSearchQueueAdapter) PublishTask(ctx context.Context, task domain.FindNewLinksTask) error {

	logger := contextkeys.LoggerFromContext(ctx)
	adapterLogger := logger.WithFields(port.Fields{
		"component":   "RabbitMQLinksSearchQueueAdapter",
		"routing_key": task.RoutingKey,
		"task_id":     task.Task.TaskID.String(),
		"category":    task.Task.Category,
		"region":      task.Task.Region,
	})

	taskJSON, err := json.Marshal(task.Task)
	if err != nil {
		adapterLogger.Error("Failed to marshal task to JSON", err, nil)
		return fmt.Errorf("rabbitmq adapter: failed to marshal task to JSON for %s - %s: %w", task.Task.Category, task.Task.Region, err)
	}

	msg := amqp.Publishing{
		ContentType:  "application/json", // Указываем, что отправляем JSON
		Body:         taskJSON,
		DeliveryMode: amqp.Persistent, // Для сохранения сообщений при перезапуске брокера
		Timestamp:    time.Now(),
		Priority: 	  task.Priority,
		Headers:      make(amqp.Table),
	}

	// Извлекаем trace_id из контекста и кладем в заголовки
	traceID := contextkeys.TraceIDFromContext(ctx)
	if traceID != "" {
		msg.Headers["x-trace-id"] = traceID
	}

	publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Таймаут 10 секунд на публикацию
	defer cancel()

	adapterLogger.Debug("Publishing new links search task", nil)
	err = a.producer.Publish(publishCtx, task.RoutingKey, msg)
	if err != nil {
		adapterLogger.Error("Failed to publish new links search task", err, nil)
		return err
	}

	adapterLogger.Debug("Successfully published task", nil)
	return nil
}
