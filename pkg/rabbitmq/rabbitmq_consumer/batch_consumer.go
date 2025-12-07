package rabbitmq_consumer

import (
	"context"
	"fmt"
	"real-estate-system/pkg/rabbitmq/rabbitmq_common"

	// "log"

	// "sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// BatchMessageHandler - обработчик для пачки сообщений.
// Он принимает срез доставок и должен вернуть ошибку, если пачку нужно вернуть в очередь.
type BatchMessageHandler func(deliveries []amqp.Delivery) error

// BatchConsumer - структура для управления пакетным потребителем.
type BatchConsumer struct {
	baseConsumer *baseConsumer
	handler      BatchMessageHandler
	batchSize    int
	batchTimeout time.Duration
}

// NewBatchConsumer создает нового пакетного потребителя.
func NewBatchConsumer(cfg ConsumerConfig, handler BatchMessageHandler, batchSize int, batchTimeout time.Duration, connManager *rabbitmq_common.ConnectionManager) (*BatchConsumer, error) {

	if cfg.PrefetchCount < batchSize {
		// log.Printf("Batch Consumer: Warning: PrefetchCount (%d) is less than BatchSize (%d). Setting PrefetchCount to BatchSize.", cfg.PrefetchCount, batchSize)

		cfg.PrefetchCount = batchSize
	}

	bc, err := newBaseConsumer(cfg, connManager)
	if err != nil {
		return nil, fmt.Errorf("batch Consumer: %w", err)
	}

	if handler == nil {
		return nil, fmt.Errorf("batch Consumer: message handler is required")
	}

	c := &BatchConsumer{
		baseConsumer: bc,
		handler:      handler,
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
	}

	return c, nil

}

// StartConsuming начинает потребление и накопление сообщений.
func (c *BatchConsumer) StartConsuming(ctx context.Context) error {
	if c.baseConsumer.channel == nil || c.baseConsumer.connection.IsClosed() {
		return fmt.Errorf("batch Consumer: not connected")
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
		return fmt.Errorf("batch Consumer: failed to register a consumer: %w", err)
	}

	c.baseConsumer.Logger.Info("[*] Waiting for messages on queue",
		"queue_name", c.baseConsumer.actualQueueName,
		"batch_size", c.batchSize,
		"batch_timeout", c.batchTimeout)

	c.baseConsumer.wg.Add(1)
	go func() {
		defer c.baseConsumer.wg.Done()
		batch := make([]amqp.Delivery, 0, c.batchSize)
		// Создаем таймер, но пока не запускаем его
		timer := time.NewTimer(c.batchTimeout)
		// Важно: нужно сразу его остановить и слить канал, чтобы он не сработал преждевременно.
		if !timer.Stop() {
			<-timer.C
		}

		for {
			select {
			case <-ctx.Done():
				// Контекст отменен. Обрабатываем последнюю собранную пачку и выходим.
				c.baseConsumer.Logger.Info("Context cancelled. Processing final batch...")
				c.processBatch(batch)
				return

			case msg, ok := <-msgs:
				if !ok {
					c.baseConsumer.Logger.Info("Deliveries channel closed. Processing final batch...")
					c.processBatch(batch)
					return
				}

				// Если это первое сообщение в новой пачке, запускаем таймер.
				if len(batch) == 0 {
					timer.Reset(c.batchTimeout)
				}

				batch = append(batch, msg)

				// Если пачка заполнилась, обрабатываем ее немедленно.
				if len(batch) >= c.batchSize {

					c.baseConsumer.Logger.Info("Batch size reached. Processing...",
						"batch_size", len(batch))

					// Останавливаем таймер, так как он нам больше не нужен для этой пачки.
					if !timer.Stop() {
						<-timer.C
					}
					c.processBatch(batch)
					batch = make([]amqp.Delivery, 0, c.batchSize) // Создаем новую пустую пачку.
				}

			case <-timer.C:
				// Таймер сработал. Обрабатываем то, что успело накопиться.
				if len(batch) > 0 {
					c.baseConsumer.Logger.Info("Timeout reached. Processing batch of messages",
						"batch_size", len(batch))
					c.processBatch(batch)
					batch = make([]amqp.Delivery, 0, c.batchSize) // Создаем новую пустую пачку.
				}
			}
		}
	}()

	// Ждем, пока соединение не будет закрыто
	notifyClose := make(chan *amqp.Error)
	c.baseConsumer.connection.NotifyClose(notifyClose)

	select {
	case <-ctx.Done():
		c.baseConsumer.Logger.Info("Context cancelled for consumer. Shutting down.",
			"consumer_tag", c.baseConsumer.config.ConsumerTag)
		return nil

	case err := <-notifyClose:
		c.baseConsumer.Logger.Error(err, "Connection closed for consumer",
			"consumer_tag", c.baseConsumer.config.ConsumerTag)
		return err
	}
}

// processBatch вызывает внешний обработчик и отправляет Ack/Nack.
func (c *BatchConsumer) processBatch(batch []amqp.Delivery) {
	if len(batch) == 0 {
		return
	}

	if err := c.handler(batch); err == nil {
		// Успех, подтверждаем всю пачку одним махом.
		lastTag := batch[len(batch)-1].DeliveryTag
		_ = c.baseConsumer.channel.Ack(lastTag, true)

		c.baseConsumer.Logger.Info("Successfully Ack'd batch of messages",
			"batch_size", len(batch))

		return
	} else {
		// Ошибка при обработке пачки
		c.baseConsumer.Logger.Error(err, "Handler returned error for batch")
	}

	if !c.baseConsumer.config.EnableRetryMechanism {
		// Ретраи выключены, просто Nack'аем всю пачку без requeue
		lastTag := batch[len(batch)-1].DeliveryTag
		_ = c.baseConsumer.channel.Nack(lastTag, true, false) // multiple=true, requeue=false
		c.baseConsumer.Logger.Info("Retry disabled. Nacking entire batch without requeue.")
		return
	}

	// Ретраи включены. Проверяем каждое сообщение индивидуально.
	for _, d := range batch {
		deathCount := c.baseConsumer.getDeathCount(d, c.baseConsumer.actualQueueName)
		if deathCount < int64(c.baseConsumer.config.MaxRetries) {
			// Лимит не достигнут, возвращаем на ретрай
			c.baseConsumer.Logger.Info("Nacking message for retry",
				"delivery_tag", d.DeliveryTag,
				"death_count", deathCount)

			_ = c.baseConsumer.channel.Nack(d.DeliveryTag, false, false) // single, requeue=false -> to DLX/retry-loop
		} else {
			// Лимит достигнут, отправляем в финальный DLQ
			c.baseConsumer.Logger.Info("Max retries reached for message. Publishing to final DLX.",
				"delivery_tag", d.DeliveryTag)

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
				c.baseConsumer.Logger.Error(err, "Failed to publish to final DLX. Nacking to trigger retry loop again.",
					"consumer_tag", c.baseConsumer.config.ConsumerTag,
					"delivery_tag", d.DeliveryTag)
				_ = d.Nack(false, false) // Пытаемся еще раз, раз не смогли отправить в DLQ
			} else {
				// Успешно опубликовали, теперь подтверждаем ОРИГИНАЛ
				c.baseConsumer.Logger.Info("Successfully published to final DLX. Acking original message",
					"consumer_tag", c.baseConsumer.config.ConsumerTag,
					"delivery_tag", d.DeliveryTag)
				_ = d.Ack(false)
			}

		}
	}

	// if err := c.handler(batch); err != nil {
	// 	log.Printf("Batch Consumer: Handler returned error for batch: %v. Nacking all %d messages with requeue.", err, len(batch))
	// 	// Вся пачка не обработалась, возвращаем все сообщения в очередь.
	// 	// Nack'аем по одному, так как RabbitMQ не поддерживает multiple Nack с requeue.
	// 	for _, d := range batch {
	// 		// Используем c.baseConsumer.channel для Nack
	// 		if nackErr := c.baseConsumer.channel.Nack(d.DeliveryTag, false, true); nackErr != nil {
	//             log.Printf("Batch Consumer: Failed to send Nack for tag %d: %v", d.DeliveryTag, nackErr)
	//         }
	// 	}
	// } else {
	// 	// Успех, подтверждаем всю пачку одним махом.
	// 	lastTag := batch[len(batch)-1].DeliveryTag

	// 	// Используем c.baseConsumer.channel для Ack
	// 	if ackErr := c.baseConsumer.channel.Ack(lastTag, true); ackErr != nil { // multiple = true
	// 		log.Printf("Batch Consumer: Failed to send multiple Ack for tag %d: %v", lastTag, ackErr)
	// 	} else {
	// 		log.Printf("Batch Consumer: Successfully Ack'd batch of %d messages up to tag %d.", len(batch), lastTag)
	// 	}
	// }
}

// Close дожидается завершения обработки последней пачки и закрывает соединение.
func (c *BatchConsumer) Close() error {
	c.baseConsumer.Logger.Info("Closing consumer")

	return c.baseConsumer.Close()
}
