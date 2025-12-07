package rabbitmq_adapter

import (
	"context"
	"encoding/json"

	// "log"
	"real-estate-system/pkg/rabbitmq/rabbitmq_common"
	"real-estate-system/pkg/rabbitmq/rabbitmq_consumer"
	"task-service/internal/contextkeys"
	"task-service/internal/core/port"
	"task-service/internal/core/port/usecases_port"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// DTO для сообщения от storage-service
type TaskResultDTO struct {
	TaskID  uuid.UUID      `json:"task_id"`
	Results map[string]int `json:"results"`
}

// ResultsConsumerAdapter - консьюмер для результатов задач.
type ResultsConsumerAdapter struct {
	consumer rabbitmq_consumer.Consumer // Ваша обертка для консьюмера
	useCase  usecases_port.ProcessTaskResultUseCasePort
	logger   port.LoggerPort
}

// NewResultsConsumerAdapter - конструктор.
func NewResultsConsumerAdapter(
	cfg rabbitmq_consumer.ConsumerConfig,
	uc usecases_port.ProcessTaskResultUseCasePort,
	logger port.LoggerPort,
	connManager *rabbitmq_common.ConnectionManager,
) (*ResultsConsumerAdapter, error) {
	adapter := &ResultsConsumerAdapter{useCase: uc, logger: logger}

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

// messageHandler - обработчик одного сообщения.
func (a *ResultsConsumerAdapter) messageHandler(d amqp.Delivery) error {
	// 1. ИЗВЛЕКАЕМ ИЛИ ГЕНЕРИРУЕМ TRACE_ID
	traceID, ok := d.Headers["x-trace-id"].(string)
	if !ok || traceID == "" {
		traceID = uuid.New().String()
	}

	// 2. СОЗДАЕМ КОНТЕКСТНЫЙ ЛОГГЕР
	msgLogger := a.logger.WithFields(port.Fields{
		"trace_id":     traceID,
		"delivery_tag": d.DeliveryTag,
	})

	// 3. ПОМЕЩАЕМ ЛОГГЕР И TRACE_ID В КОНТЕКСТ
	ctx := context.Background()
	ctx = contextkeys.ContextWithTraceID(ctx, traceID)

	var dto TaskResultDTO
	if err := json.Unmarshal(d.Body, &dto); err != nil {
		msgLogger.Error("Failed to unmarshal task result DTO, rejecting message.", err, nil)
		return nil // Не переотправляем "битые" сообщения
	}

	handlerLogger := msgLogger.WithFields(port.Fields{
		"task_id": dto.TaskID.String(),
	})
	// Обновляем контекст с более детальным логгером
	ctx = contextkeys.ContextWithLogger(ctx, handlerLogger)

	handlerLogger.Info("Processing task result.", port.Fields{"results": dto.Results})

	// Вызываем Use Case для инкрементации счетчиков
	if err := a.useCase.Execute(ctx, dto.TaskID, dto.Results); err != nil {
		handlerLogger.Error("Failed to process task result, message will be nacked for retry.", err, nil)
		return err // Возвращаем ошибку, чтобы RabbitMQ попробовал снова
	}

	handlerLogger.Info("Successfully processed task result.", nil)
	return nil
}

// Start и Close методы, которые делегируют вызовы a.consumer.
func (a *ResultsConsumerAdapter) Start(ctx context.Context) error {
	return a.consumer.StartConsuming(ctx)
}
func (a *ResultsConsumerAdapter) Close() error { return a.consumer.Close() }
