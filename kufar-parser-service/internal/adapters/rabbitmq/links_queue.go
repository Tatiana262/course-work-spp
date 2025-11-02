package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"kufar-parser-service/internal/core/domain"
	"kufar-parser-service/pkg/rabbitmq/rabbitmq_producer"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQLinkQueueAdapter реализует интерфейс PropertyLinkQueuePort для RabbitMQ
type RabbitMQLinkQueueAdapter struct {
	producer   *rabbitmq_producer.Publisher
	routingKey string 
}

// NewRabbitMQLinkQueueAdapter создает новый экземпляр RabbitMQLinkQueueAdapter
// producer - это экземпляр rabbitmq_producer.Publisher
// routingKey - ключ, с которым будут публиковаться сообщения 
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

// Enqueue отправляет ссылку в очередь RabbitMQ
func (a *RabbitMQLinkQueueAdapter) Enqueue(ctx context.Context, link domain.PropertyLink) error {

	linkJSON, err := json.Marshal(link)
	if err != nil {
		return fmt.Errorf("rabbitmq adapter: failed to marshal property link to JSON for AdID %d: %w", link.AdID, err)
	}

	msg := amqp.Publishing{
		ContentType:  "application/json",
		Body:         linkJSON,
		DeliveryMode: amqp.Persistent, 
		Timestamp:    time.Now(),
	}

	publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Таймаут 10 секунд на публикацию
	defer cancel()

	log.Printf("RabbitMQAdapter: Publishing link with AdID to routing key '%s': %d\n", a.routingKey, link.AdID)
	err = a.producer.Publish(publishCtx, a.routingKey, msg)
	if err != nil {
		return fmt.Errorf("rabbitmq adapter: failed to publish link with AdID %d: %w", link.AdID, err)
	}

	log.Printf("RabbitMQAdapter: Successfully published link with AdID: %d\n", link.AdID)
	return nil
}