package rabbitmq_consumer

import (
	"context"
	"fmt"
	"log"

	// "sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// BatchMessageHandler - обработчик для пачки сообщений
type BatchMessageHandler func(deliveries []amqp.Delivery) error


// BatchConsumer - структура для управления пакетным потребителем
type BatchConsumer struct {
	baseConsumer *baseConsumer 
	handler BatchMessageHandler
	batchSize      int
	batchTimeout   time.Duration
}

// NewBatchConsumer создает нового пакетного потребителя
func NewBatchConsumer(cfg ConsumerConfig, handler BatchMessageHandler, batchSize int, batchTimeout time.Duration) (*BatchConsumer, error) {
	
	if cfg.PrefetchCount < batchSize {
		log.Printf("Batch Consumer: Warning: PrefetchCount (%d) is less than BatchSize (%d). Setting PrefetchCount to BatchSize.", cfg.PrefetchCount, batchSize)
		cfg.PrefetchCount = batchSize
	}

	bc, err := newBaseConsumer(cfg)
	if err != nil {
		return nil, fmt.Errorf("Batch Consumer: %w", err)
	}

	if handler == nil {
		return nil, fmt.Errorf("Batch Consumer: message handler is required")
	}

	c := &BatchConsumer{
		baseConsumer:  bc,
		handler: handler,
		batchSize: batchSize,
		batchTimeout: batchTimeout,
	}

	return c, nil
	
}

// StartConsuming начинает потребление и накопление сообщений
func (c *BatchConsumer) StartConsuming(ctx context.Context) error {
	if c.baseConsumer.channel == nil || c.baseConsumer.connection.IsClosed() {
		return fmt.Errorf("Batch Consumer: not connected")
	}

	msgs, err := c.baseConsumer.channel.Consume(
		c.baseConsumer.actualQueueName,
		c.baseConsumer.config.ConsumerTag,
		false, // auto-ack = false
		c.baseConsumer.config.ExclusiveConsumer,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("Batch Consumer: failed to register a consumer: %w", err)
	}

	log.Printf("Batch Consumer: [*] Waiting for messages on queue '%s'. BatchSize: %d, Timeout: %s\n", c.baseConsumer.actualQueueName, c.batchSize, c.batchTimeout)

	c.baseConsumer.wg.Add(1)
	go func() {
		defer c.baseConsumer.wg.Done()
		batch := make([]amqp.Delivery, 0, c.batchSize)

		timer := time.NewTimer(c.batchTimeout)
		// чтобы не сработал преждевременно
		if !timer.Stop() {
			<-timer.C
		}

		for {
			select {
			case <-ctx.Done():
				// Контекст отменен. Обрабатываем последнюю собранную пачку и выходим
				log.Println("Batch Consumer: Context cancelled. Processing final batch...")
				c.processBatch(batch)
				return

			case msg, ok := <-msgs:
				if !ok {
					log.Println("Batch Consumer: Deliveries channel closed. Processing final batch...")
					c.processBatch(batch)
					return
				}

				// Если это первое сообщение в новой пачке, запускаем таймер
				if len(batch) == 0 {
					timer.Reset(c.batchTimeout)
				}
				
				batch = append(batch, msg)

				// Если пачка заполнилась, обрабатываем ее немедленно
				if len(batch) >= c.batchSize {
					log.Printf("Batch Consumer: Batch size reached (%d). Processing...", len(batch))
					// Останавливаем таймер
					if !timer.Stop() {
						<-timer.C
					}
					c.processBatch(batch)
					batch = make([]amqp.Delivery, 0, c.batchSize) // Создаем новую пустую пачку
				}

			case <-timer.C:
				// Таймер сработал. Обрабатываем то, что успело накопиться
				if len(batch) > 0 {
					log.Printf("Batch Consumer: Timeout reached. Processing batch of %d messages...", len(batch))
					c.processBatch(batch)
					batch = make([]amqp.Delivery, 0, c.batchSize) // Создаем новую пустую пачку
				}
			}
		}
	}()

	// Ждем, пока соединение не будет закрыто
	notifyClose := make(chan *amqp.Error)
	c.baseConsumer.connection.NotifyClose(notifyClose)

	select {
	case <-ctx.Done():
		log.Printf("Batch Consumer: Context cancelled for tag '%s'. Shutting down.", c.baseConsumer.config.ConsumerTag)
		return nil

	case err := <-notifyClose:
		log.Printf("Batch Consumer: Connection closed for tag '%s'. Error: %v", c.baseConsumer.config.ConsumerTag, err)
		return err
	}
}

// processBatch вызывает внешний обработчик и отправляет Ack/Nack
func (c *BatchConsumer) processBatch(batch []amqp.Delivery) {
	if len(batch) == 0 {
		return
	}

	if err := c.handler(batch); err == nil {
		// Успех, подтверждаем всю пачку
		lastTag := batch[len(batch)-1].DeliveryTag
		_ = c.baseConsumer.channel.Ack(lastTag, true)
		log.Printf("Batch Consumer: Successfully Ack'd batch of %d messages.", len(batch))
		return
	} else {
		// Ошибка при обработке пачки
		log.Printf("Batch Consumer: Handler returned error for batch: %v.", err)
	}

	

	if !c.baseConsumer.config.EnableRetryMechanism {
		// Ретраи выключены, просто Nack всей пачки без requeue
		lastTag := batch[len(batch)-1].DeliveryTag
		_ = c.baseConsumer.channel.Nack(lastTag, true, false) // multiple=true, requeue=false
		log.Println("Batch Consumer: Retry disabled. Nacking entire batch without requeue.")
		return
	}

	// Ретраи включены, проверяем каждое сообщение
	for _, d := range batch {
		deathCount := c.baseConsumer.getDeathCount(d, c.baseConsumer.actualQueueName)
		if deathCount < int64(c.baseConsumer.config.MaxRetries) {
			// Лимит не достигнут, возвращаем на ретрай
			log.Printf("Batch Consumer: Nacking message (Tag: %d) for retry, death count: %d", d.DeliveryTag, deathCount)
			_ = c.baseConsumer.channel.Nack(d.DeliveryTag, false, false) // single, requeue=false
		} else {
			// Лимит достигнут, отправляем в финальный DLQ
			log.Printf("Batch Consumer: Max retries reached for message (Tag: %d). Publishing to final DLX.", d.DeliveryTag)

			err := c.baseConsumer.finalDlxPublisher.Publish(
				context.Background(),
				c.baseConsumer.config.FinalDLQRoutingKey,
				amqp.Publishing{
					ContentType:  d.ContentType,
					Body:         d.Body,
					Headers:      d.Headers,
					Timestamp:    time.Now(),
					DeliveryMode: amqp.Persistent,
				},
			)

			if err != nil {
				log.Printf("Batch Consumer: FAILED to publish to final DLX: %v. Nacking to trigger retry loop again.", err)
				_ = d.Nack(false, false) 
			} else {
				log.Printf("Batch Consumer: Successfully published to final DLX. Acking original message (Tag: %d).", d.DeliveryTag)
				_ = d.Ack(false)
			}
		
		}
	}

}

// Close дожидается завершения обработки последней пачки и закрывает соединение
func (c *BatchConsumer) Close() error {
	log.Println("Batch Consumer: Closing...")
	
	return c.baseConsumer.Close()
}