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

// TaskManagerPublisher - реализация порта TaskResultsPort для RabbitMQ
type TaskManagerPublisher struct {
	producer   *rabbitmq_producer.Publisher
	routingKey string
}

// NewTaskManagerPublisher - конструктор
func NewTaskManagerPublisher(producer *rabbitmq_producer.Publisher, routingKey string) (*TaskManagerPublisher, error) {
	if producer == nil {
		return nil, fmt.Errorf("rabbitmq adapter: producer cannot be nil")
	}
	if routingKey == "" {
		return nil, fmt.Errorf("rabbitmq adapter: routingKey cannot be empty")
	}
	return &TaskManagerPublisher{
		producer:   producer,
		routingKey: routingKey,
	}, nil
}


func (a *TaskManagerPublisher) PublishCompletionCommand(ctx context.Context, cmd domain.TaskCompletionCommand) error {

	logger := contextkeys.LoggerFromContext(ctx)
	adapterLogger := logger.WithFields(port.Fields{
		"component":   "TaskManagerPublisher",
		"routing_key": a.routingKey,
		"task_id":     cmd.TaskID.String(),
	})

	body, err := json.Marshal(cmd)
	if err != nil {
		adapterLogger.Error("Failed to marshal completion command", err, nil)
		return fmt.Errorf("failed to marshal completion command: %w", err)
	}

	msg := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent, // Для сохранения сообщений при перезапуске брокера
		Timestamp:    time.Now(),
		Headers:      make(amqp.Table),
	}

	// Извлекаем trace_id из контекста и кладем в заголовки
	traceID := contextkeys.TraceIDFromContext(ctx)
	if traceID != "" {
		msg.Headers["x-trace-id"] = traceID
	}

	publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Таймаут 10 секунд на публикацию
	defer cancel()

	adapterLogger.Debug("Publishing completion command", nil)
	err = a.producer.Publish(publishCtx, a.routingKey, msg)
	if err != nil {
		adapterLogger.Error("Failed to publish completion command", err, nil)
		return fmt.Errorf("rabbitmq adapter: failed to publish completion command for task %s: %w", cmd.TaskID, err)
	}

	adapterLogger.Info("Successfully published completion command", port.Fields{"expected_results": cmd.Results})
	return nil
}
