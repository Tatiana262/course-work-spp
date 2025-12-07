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

// DTO для сообщения из task_management_queue.
type TaskManagementCommandDTO struct {
	TaskID               uuid.UUID `json:"task_id"`
	ExpectedResultsCount int       `json:"expected_results_count"`
}

// ManagementConsumerAdapter - слушает управляющие команды для задач.
type TaskCompletionResultsConsumerAdapter struct {
	consumer rabbitmq_consumer.Consumer            // Ваша обертка
	useCase  usecases_port.CompleteTaskUseCasePort // Новый Use Case
	logger   port.LoggerPort
}

// NewManagementConsumerAdapter - конструктор.
func NewTaskCompletionResultsConsumerAdapter(
	cfg rabbitmq_consumer.ConsumerConfig,
	uc usecases_port.CompleteTaskUseCasePort,
	logger port.LoggerPort,
	connManager *rabbitmq_common.ConnectionManager,
) (*TaskCompletionResultsConsumerAdapter, error) {

	adapter := &TaskCompletionResultsConsumerAdapter{useCase: uc, logger: logger}

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

// messageHandler обрабатывает одну управляющую команду.
func (a *TaskCompletionResultsConsumerAdapter) messageHandler(d amqp.Delivery) error {
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

	var dto TaskManagementCommandDTO
	if err := json.Unmarshal(d.Body, &dto); err != nil {
		msgLogger.Error("Failed to unmarshal management command DTO, rejecting message.", err, nil)
		return nil // "Битое" сообщение, не переотправляем.
	}

	handlerLogger := msgLogger.WithFields(port.Fields{
		"task_id": dto.TaskID.String(),
	})
	ctx = contextkeys.ContextWithLogger(ctx, handlerLogger)

	handlerLogger.Info("Processing completion command.", port.Fields{"expected_results": dto.ExpectedResultsCount})

	// Вызываем Use Case
	if err := a.useCase.Execute(ctx, dto.TaskID, dto.ExpectedResultsCount); err != nil {
		handlerLogger.Error("Failed to process completion command, retrying.", err, nil)
		return err // Возвращаем ошибку для механизма retry
	}

	handlerLogger.Info("Successfully processed completion command.", nil)
	return nil
}

// Start и Close
func (a *TaskCompletionResultsConsumerAdapter) Start(ctx context.Context) error {
	return a.consumer.StartConsuming(ctx)
}
func (a *TaskCompletionResultsConsumerAdapter) Close() error { return a.consumer.Close() }
