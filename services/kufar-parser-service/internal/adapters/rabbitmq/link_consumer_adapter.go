package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"kufar-parser-service/internal/contextkeys"
	"kufar-parser-service/internal/core/domain"
	"kufar-parser-service/internal/core/port"
	usecases_port "kufar-parser-service/internal/core/port/usecases"

	// "kufar-parser-service/internal/core/usecase"
	"real-estate-system/pkg/rabbitmq/rabbitmq_common"
	"real-estate-system/pkg/rabbitmq/rabbitmq_consumer"

	// "log"

	// "strings"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// LinkConsumerAdapter - это входящий адаптер, который слушает очередь
// со ссылками и вызывает use case для их обработки.
type LinkConsumerAdapter struct {
	consumer rabbitmq_consumer.Consumer
	useCase  usecases_port.ProcessLinkPort // Зависимость от конкретного UseCase
	logger   port.LoggerPort
}

// NewLinkConsumerAdapter создает новый адаптер.
func NewLinkConsumerAdapter(
	consumerCfg rabbitmq_consumer.ConsumerConfig,
	useCase usecases_port.ProcessLinkPort,
	logger port.LoggerPort,
	connManager *rabbitmq_common.ConnectionManager,
) (*LinkConsumerAdapter, error) {

	adapter := &LinkConsumerAdapter{
		useCase: useCase,
		logger:  logger,
	}

	// 1. Создаем логгер для pkg-уровня с контекстом нашего компонента
	pkgLogger := logger.WithFields(port.Fields{"component": "rabbitmq_distributing_consumer", "consumer_tag": consumerCfg.ConsumerTag})
	consumerCfg.Logger = NewPkgLoggerBridge(pkgLogger)

	// Создаем consumer, передавая ему метод этого адаптера как обработчик.
	// Теперь `messageHandler` является частью адаптера, а не App.
	consumer, err := rabbitmq_consumer.NewDistributingConsumer(consumerCfg, adapter.messageHandler, connManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ consumer for links: %w", err)
	}
	adapter.consumer = consumer

	return adapter, nil
}

// messageHandler - приватный метод адаптера.
func (a *LinkConsumerAdapter) messageHandler(d amqp.Delivery) (err error) {
	// 1. ИЗВЛЕКАЕМ ИЛИ ГЕНЕРИРУЕМ TRACE_ID
	traceID, ok := d.Headers["x-trace-id"].(string)
	if !ok || traceID == "" {
		traceID = uuid.New().String()
	}

	// 2. СОЗДАЕМ КОНТЕКСТНЫЙ ЛОГГЕР
	msgLogger := a.logger.WithFields(port.Fields{
		"trace_id":     traceID,
		"delivery_tag": d.DeliveryTag,
		"consumer_tag": d.ConsumerTag,
	})

	// 3. ПОМЕЩАЕМ ЛОГГЕР И TRACE_ID В КОНТЕКСТ
	ctx := context.Background()
	ctx = contextkeys.ContextWithLogger(ctx, msgLogger)
	ctx = contextkeys.ContextWithTraceID(ctx, traceID)

	msgLogger.Info("Received new link task", nil)

	var taskDTO LinkTaskDTO
	if err := json.Unmarshal(d.Body, &taskDTO); err != nil {
		msgLogger.Error("Error unmarshalling DTO, NACKing message", err, nil)
		return fmt.Errorf("unmarshal DTO error: %w", err)
	}

	// Обогащаем логгер данными из задачи
	taskLogger := msgLogger.WithFields(port.Fields{
		"ad_id":   taskDTO.AdID,
		"task_id": taskDTO.TaskID.String(),
	})
	// Обновляем контекст с более детальным логгером
	ctx = contextkeys.ContextWithLogger(ctx, taskLogger)

	linkToParse := domain.PropertyLink{
		AdID:   taskDTO.AdID,
		Source: taskDTO.Source,
	}

	// Адаптер вызывает UseCase
	err = a.useCase.Execute(ctx, linkToParse, taskDTO.TaskID)
	if err != nil {
		taskLogger.Error("Use case failed with a potentially transient error, requeueing", err, nil)
		return err // Requeue=true
	}

	taskLogger.Info("Link task processed successfully", nil)
	return nil
}

// Start реализует EventListenerPort
func (a *LinkConsumerAdapter) Start(ctx context.Context) error {
	return a.consumer.StartConsuming(ctx)
}

// Close реализует EventListenerPort
func (a *LinkConsumerAdapter) Close() error {
	return a.consumer.Close()
}
