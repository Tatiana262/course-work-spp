package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"realt-parser-service/internal/core/domain"
	usecases_port "realt-parser-service/internal/core/port/usecases"
	"realt-parser-service/internal/core/usecase"
	"realt-parser-service/pkg/rabbitmq/rabbitmq_consumer"
	"log"


	amqp "github.com/rabbitmq/amqp091-go"
)

// LinkConsumerAdapter - это входящий адаптер, который слушает очередь
// со ссылками и вызывает use case для их обработки
type LinkConsumerAdapter struct {
	consumer   rabbitmq_consumer.Consumer
	useCase    usecases_port.ProcessLinkPort
}

// NewLinkConsumerAdapter создает новый адаптер
func NewLinkConsumerAdapter(
	consumerCfg rabbitmq_consumer.ConsumerConfig,
	useCase *usecase.ProcessLinkUseCase,
) (*LinkConsumerAdapter, error) {
	
	adapter := &LinkConsumerAdapter{
		useCase: useCase,
	}

	// Создаем consumer, передавая ему метод этого адаптера как обработчик
	consumer, err := rabbitmq_consumer.NewDistributingConsumer(consumerCfg, adapter.messageHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ consumer for links: %w", err)
	}
	adapter.consumer = consumer

	return adapter, nil
}


func (a *LinkConsumerAdapter) messageHandler(d amqp.Delivery) (err error) {
	log.Printf("LinkConsumerAdapter: Received task (Tag: %d)\n", d.DeliveryTag)

	var linkToParse domain.PropertyLink
	if err := json.Unmarshal(d.Body, &linkToParse); err != nil {
		log.Printf("LinkConsumerAdapter: Error unmarshalling: %v. NACK (no requeue).\n", err)
		return fmt.Errorf("unmarshal error: %w", err)
	}

	// Адаптер вызывает UseCase
	err = a.useCase.Execute(context.Background(), linkToParse)
	if err != nil {
		log.Printf("LinkConsumerAdapter: Use case failed with a potentially transient error: %v. Requeueing task.", err)
		return err
	}

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