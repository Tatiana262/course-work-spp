package rabbitmq_consumer

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// MessageHandler функция-обработчик для полученных сообщений
type MessageHandler func(delivery amqp.Delivery) error


// Consumer структура для управления потребителем
type DistributingConsumer struct {
	baseConsumer *baseConsumer 
	handler    MessageHandler
	
}

// NewConsumer создает нового потребителя
func NewDistributingConsumer(cfg ConsumerConfig, handler MessageHandler) (*DistributingConsumer, error) {
	
	bc, err := newBaseConsumer(cfg)
	if err != nil {
		return nil, fmt.Errorf("Distributing Consumer: %w", err)
	}

	if handler == nil {
		return nil, fmt.Errorf("Distributing Consumer: message handler is required")
	}

	c := &DistributingConsumer{
		baseConsumer:  bc,
		handler: handler,
	}

	return c, nil
}


// StartConsuming начинает потребление сообщений
func (c *DistributingConsumer) StartConsuming(ctx context.Context) error {
	if c.baseConsumer.channel == nil || c.baseConsumer.connection == nil || c.baseConsumer.connection.IsClosed() {
		return fmt.Errorf("Distributing Consumer: not connected. Please create a new consumer or ensure connection is stable")
	}


	msgs, err := c.baseConsumer.channel.Consume(
		c.baseConsumer.actualQueueName,     // Используем актуальное имя очереди
		c.baseConsumer.config.ConsumerTag,
		false,                 // auto-ack
		c.baseConsumer.config.ExclusiveConsumer, 
		false,                 // no-local
		false,                 // no-wait
		nil,                   // args
	)
	if err != nil {
		return fmt.Errorf("Distributing Consumer: failed to register a consumer on queue '%s': %w", c.baseConsumer.actualQueueName, err)
	}

	log.Printf("Distributing Consumer: [*] Waiting for messages on queue '%s'.\n", c.baseConsumer.actualQueueName)

	// Запускаем горутину, которая будет читать из канала RabbitMQ и распределять работу
	go func() {
		for {
			// Приоритетная, неблокирующая проверка на отмену (гарантирует, что мы не запустим нового работника, если уже получили команду на остановку)
			select {
			case <-ctx.Done():
				log.Printf("Distributing Consumer: (Priority Check) Context cancelled for tag '%s'. Exiting consumption loop.", c.baseConsumer.config.ConsumerTag)
				return // Выход из основной горутины
			default:
				// Контекст не отменен, продолжаем
			}

			// Блокирующее ожидание нового сообщения или отмены
			select {
			case <-ctx.Done(): // Если контекст был отменен, пока мы ждали сообщение
				log.Printf("Distributing Consumer: (Wait Check) Context cancelled for tag '%s'. Exiting consumption loop.", c.baseConsumer.config.ConsumerTag)
				return // Выход из основной горутины
				
			case d, ok := <-msgs:
				if !ok {
					log.Printf("Distributing Consumer: Deliveries channel closed by RabbitMQ for tag '%s'. Exiting loop.", c.baseConsumer.config.ConsumerTag)
					return
				}
				
				// Запускаем обработчик для каждого сообщения в новой горутине
				c.baseConsumer.wg.Add(1) 
				go func(delivery amqp.Delivery) {
					defer c.baseConsumer.wg.Done()

					log.Printf("Distributing Consumer: [->] Started processing message (Tag: %d)\n", delivery.DeliveryTag)
					
					processErr := c.handler(delivery) 

					if processErr == nil {
						_ = delivery.Ack(false)
						log.Printf("Distributing Consumer: [+] Message Ack'd (Tag: %d)\n", delivery.DeliveryTag)
						return
					}
					
					log.Printf("Distributing Consumer: Handler error for message (Tag: %d): %v", delivery.DeliveryTag, processErr)
					
					if !c.baseConsumer.config.EnableRetryMechanism {
						log.Println("Distributing Consumer: Retry disabled. Nacking message without requeue.")
						_ = delivery.Nack(false, false)
						return
					}

					// сколько раз сообщение уже умирало
					deathCount := c.baseConsumer.getDeathCount(delivery, c.baseConsumer.actualQueueName)

					if deathCount < int64(c.baseConsumer.config.MaxRetries) {
						// Лимит не достигнут, отправляем в цикл ретрая через Nack(requeue=false)
						log.Printf("Distributing Consumer: Retrying message (Tag: %d), death count: %d", delivery.DeliveryTag, deathCount)
						_ = delivery.Nack(false, false)
					} else {
						// Лимит ретраев исчерпан, публикуем в финальный DLX
						log.Printf("Distributing Consumer: Max retries reached for message (Tag: %d). Publishing to final DLX.", delivery.DeliveryTag)
						
						err := c.baseConsumer.finalDlxPublisher.Publish(
							context.Background(),
							c.baseConsumer.config.FinalDLQRoutingKey,
							amqp.Publishing{
								ContentType:  delivery.ContentType,
								Body:         delivery.Body,
								Headers:      delivery.Headers,
								Timestamp:    time.Now(),
								DeliveryMode: amqp.Persistent,
							},
						)
						
						if err != nil {
							log.Printf("Distributing Consumer: FAILED to publish to final DLX: %v. Nacking to trigger retry loop again.", err)
							_ = delivery.Nack(false, false) // Пытаемся еще раз, раз не смогли отправить в DLQ
						} else {
							// Успешно опубликовали, теперь подтверждаем оригинал
							log.Printf("Distributing Consumer: Successfully published to final DLX. Acking original message (Tag: %d).", delivery.DeliveryTag)
							_ = delivery.Ack(false)
						}
					}
				}(d)
			}
		}
	}()

	// Ждем, пока соединение не будет закрыто
	notifyClose := make(chan *amqp.Error)
	c.baseConsumer.connection.NotifyClose(notifyClose)
	
	// ждем либо отмены внешнего контекста, либо закрытия соединения
	select {
	case <-ctx.Done():
		log.Printf("Distributing Consumer: Context cancelled for tag '%s'. Shutting down consumer.", c.baseConsumer.config.ConsumerTag)
		return nil

	case err := <-notifyClose:
		log.Printf("Distributing Consumer: Connection closed for tag '%s'. Error: %v", c.baseConsumer.config.ConsumerTag, err)
		return err 
	}
}


// Close закрывает соединение потребителя
func (c *DistributingConsumer) Close() error {
	log.Println("Distributing Consumer: Closing...")

	return c.baseConsumer.Close()
}