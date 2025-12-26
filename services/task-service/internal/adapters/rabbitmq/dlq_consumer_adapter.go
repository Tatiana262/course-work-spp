package rabbitmq_adapter

import (
	"context"
	"encoding/json"
	"real-estate-system/pkg/rabbitmq/rabbitmq_common"
	"real-estate-system/pkg/rabbitmq/rabbitmq_consumer"
	"task-service/internal/contextkeys"
	"task-service/internal/core/domain"
	"task-service/internal/core/port"
	"task-service/internal/core/port/usecases_port"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type genericMessage struct {
	TaskID  uuid.UUID      `json:"task_id"`
}

type DLQConsumerAdapter struct {
	consumer rabbitmq_consumer.Consumer 
	useCase  usecases_port.UpdateTaskStatusUseCasePort
	logger   port.LoggerPort
}


func NewDLQConsumerAdapter(
	cfg rabbitmq_consumer.ConsumerConfig,
	useCase  usecases_port.UpdateTaskStatusUseCasePort,
	logger port.LoggerPort,
	connManager *rabbitmq_common.ConnectionManager,
) (*DLQConsumerAdapter, error) {
	adapter := &DLQConsumerAdapter{useCase: useCase, logger: logger}

	// 1. Создаем логгер для pkg-уровня с контекстом нашего компонента
	pkgLogger := logger.WithFields(port.Fields{"component": "rabbitmq_distributing_consumer", "consumer_tag": cfg.ConsumerTag})
	cfg.Logger = NewPkgLoggerBridge(pkgLogger)

	consumer, err := rabbitmq_consumer.NewDistributingConsumer(cfg, adapter.messageHandler, connManager)
	if err != nil {
		return nil, err
	}
	adapter.consumer = consumer
	return adapter, nil
}

func (a *DLQConsumerAdapter) messageHandler(d amqp.Delivery) error {
	// 1. ИЗВЛЕКАЕМ ИЛИ ГЕНЕРИРУЕМ TRACE_ID
	traceID, ok := d.Headers["x-trace-id"].(string)
	if !ok || traceID == "" {
		traceID = uuid.New().String()
	}

	// 2. СОЗДАЕМ КОНТЕКСТНЫЙ ЛОГГЕР
	msgLogger := a.logger.WithFields(port.Fields{
		"trace_id":     traceID,
		"delivery_tag": d.DeliveryTag,
		"queue":        d.RoutingKey, // В DLQ routing_key часто содержит имя исходной очереди
		"exchange":     d.Exchange,
	})

	var deathInfo interface{}
	if d.Headers != nil {
		if di, ok := d.Headers["x-death"]; ok {
			deathInfo = di
		}
	}

	msgLogger.Error(
		"Processing dead letter from DLQ", // Основное сообщение
		nil, // Ошибка `error` у нас нет, но мы передаем `nil`
		port.Fields{
			"body_as_string": string(d.Body), // Тело сообщения как строка
			"headers":        d.Headers,      // Все заголовки
			"x_death_info":   deathInfo,      // Информация о причине "смерти"
		},
	)

	// 3. ПОМЕЩАЕМ ЛОГГЕР И TRACE_ID В КОНТЕКСТ
	ctx := context.Background()
	ctx = contextkeys.ContextWithTraceID(ctx, traceID)

	var msg genericMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		msgLogger.Error("Failed to unmarshal genericMessage, rejecting message.", err, nil)
		return nil 
	}

	handlerLogger := msgLogger.WithFields(port.Fields{
		"task_id": msg.TaskID.String(),
	})
	// Обновляем контекст с более детальным логгером
	ctx = contextkeys.ContextWithLogger(ctx, handlerLogger)

	handlerLogger.Debug("Processing failed task", nil)

	if _, err := a.useCase.Execute(ctx, msg.TaskID, domain.StatusFailed); err != nil {
		handlerLogger.Error("Failed to process failed task, message will be nacked for retry.", err, nil)
		return err // Возвращаем ошибку, чтобы RabbitMQ попробовал снова
	}

	handlerLogger.Debug("Successfully processed failed result.", nil)
	return nil
}


func (a *DLQConsumerAdapter) Start(ctx context.Context) error {
	return a.consumer.StartConsuming(ctx)
}

func (a *DLQConsumerAdapter) Close() error { return a.consumer.Close() }