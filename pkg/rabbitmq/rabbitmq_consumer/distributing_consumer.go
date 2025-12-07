package rabbitmq_consumer

import (
	"context"
	"fmt"
	"real-estate-system/pkg/rabbitmq/rabbitmq_common"

	// "log"
	"time"

	// "parser-project/pkg/rabbitmq/rabbitmq_common"
	// "sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// MessageHandler функция-обработчик для полученных сообщений
// MessageHandler - меняем контракт на более простой, как в примере.
// Пакет сам будет решать, как делать ack/nack/requeue.
type MessageHandler func(delivery amqp.Delivery) error

// Consumer структура для управления потребителем
type DistributingConsumer struct {
	baseConsumer *baseConsumer
	handler      MessageHandler
}

// NewConsumer создает нового потребителя
func NewDistributingConsumer(cfg ConsumerConfig, handler MessageHandler, connManager *rabbitmq_common.ConnectionManager) (*DistributingConsumer, error) {

	bc, err := newBaseConsumer(cfg, connManager)
	if err != nil {
		return nil, fmt.Errorf("distributing Consumer: %w", err)
	}

	if handler == nil {
		return nil, fmt.Errorf("distributing Consumer: message handler is required")
	}

	c := &DistributingConsumer{
		baseConsumer: bc,
		handler:      handler,
	}

	return c, nil
}

// StartConsuming начинает потребление сообщений
func (c *DistributingConsumer) StartConsuming(ctx context.Context) error {
	if c.baseConsumer.channel == nil || c.baseConsumer.connection == nil || c.baseConsumer.connection.IsClosed() {
		return fmt.Errorf("distributing Consumer: not connected. Please create a new consumer or ensure connection is stable")
	}

	// Создаем контекст, который мы можем отменить, чтобы остановить горутины
	// ctx, cancel := context.WithCancel(context.Background())
	// c.cancelFunc = cancel

	msgs, err := c.baseConsumer.channel.Consume(
		c.baseConsumer.actualQueueName, // Используем актуальное имя очереди
		c.baseConsumer.config.ConsumerTag,
		false, // auto-ack
		c.baseConsumer.config.ExclusiveConsumer,
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("distributing Consumer %s: failed to register a consumer on queue '%s': %w", c.baseConsumer.config.ConsumerTag, c.baseConsumer.actualQueueName, err)
	}

	c.baseConsumer.Logger.Info("[*] Waiting for messages on queue", "queue_name", c.baseConsumer.actualQueueName)

	// Запускаем горутину, которая будет читать из канала RabbitMQ и распределять работу
	go func() {
		for {
			// --- Шаг 1: Приоритетная, неблокирующая проверка на отмену ---
			// Это гарантирует, что мы не запустим нового "работника", если уже получили команду на остановку.
			select {
			case <-ctx.Done():
				c.baseConsumer.Logger.Info("(Priority Check) Context cancelled for consumer. Exiting consumption loop.",
					"consumer_tag", c.baseConsumer.config.ConsumerTag)
				return // Выходим из горутины-диспетчера
			default:
				// Контекст не отменен, продолжаем.
			}

			// --- Шаг 2: Блокирующее ожидание нового сообщения ИЛИ отмены ---
			// Этот select ждет, пока что-то произойдет.
			select {
			case <-ctx.Done(): // Если контекст был отменен (например, при вызове Close)
				// Эта ветка сработает, если контекст отменили, ПОКА мы ждали сообщение.
				c.baseConsumer.Logger.Info("(Wait Check) Context cancelled for consumer. Exiting consumption loop.",
					"consumer_tag", c.baseConsumer.config.ConsumerTag)
				return // Выходим из горутины-диспетчера

			case d, ok := <-msgs:
				if !ok {
					c.baseConsumer.Logger.Info("Deliveries channel closed by RabbitMQ for consumer. Exiting loop.",
						"consumer_tag", c.baseConsumer.config.ConsumerTag)
					return
				}

				// Запускаем обработчик для каждого сообщения в новой горутине
				c.baseConsumer.wg.Add(1) // Увеличиваем счетчик WaitGroup
				go func(delivery amqp.Delivery) {
					defer c.baseConsumer.wg.Done() // Уменьшаем счетчик, когда горутина завершается

					c.baseConsumer.Logger.Info("[->] Started processing message",
						"consumer_tag", c.baseConsumer.config.ConsumerTag,
						"delivery_tag", delivery.DeliveryTag)

					processErr := c.handler(delivery) // Используем новый, простой обработчик

					if processErr == nil {
						// Успех! Просто подтверждаем.
						_ = delivery.Ack(false)
						c.baseConsumer.Logger.Info("[+] Message Ack'd",
							"consumer_tag", c.baseConsumer.config.ConsumerTag,
							"delivery_tag", delivery.DeliveryTag)
						return
					}

					// --- НОВАЯ ЛОГИКА ОБРАБОТКИ ОШИБОК ---

					c.baseConsumer.Logger.Error(processErr, "Handler error for message",
						"consumer_tag", c.baseConsumer.config.ConsumerTag,
						"delivery_tag", delivery.DeliveryTag)

					if !c.baseConsumer.config.EnableRetryMechanism {
						c.baseConsumer.Logger.Info("Retry disabled. Nacking message without requeue.",
							"consumer_tag", c.baseConsumer.config.ConsumerTag)
						_ = delivery.Nack(false, false)
						return
					}

					// Считаем, сколько раз сообщение уже умирало
					deathCount := c.baseConsumer.getDeathCount(delivery, c.baseConsumer.actualQueueName)

					if deathCount < int64(c.baseConsumer.config.MaxRetries) {
						// Лимит не достигнут, отправляем в цикл ретрая через Nack(requeue=false)
						c.baseConsumer.Logger.Info("Retrying message",
							"consumer_tag", c.baseConsumer.config.ConsumerTag,
							"delivery_tag", delivery.DeliveryTag,
							"death_count", deathCount)
						_ = delivery.Nack(false, false)
					} else {
						// Лимит ретраев исчерпан, публикуем в финальный DLX
						c.baseConsumer.Logger.Info("Max retries reached for message. Publishing to final DLX.",
							"consumer_tag", c.baseConsumer.config.ConsumerTag,
							"delivery_tag", delivery.DeliveryTag)

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
							c.baseConsumer.Logger.Error(err, "Failed to publish to final DLX. Nacking to trigger retry loop again.",
								"consumer_tag", c.baseConsumer.config.ConsumerTag,
								"delivery_tag", delivery.DeliveryTag)
							_ = delivery.Nack(false, false) // Пытаемся еще раз, раз не смогли отправить в DLQ
						} else {
							// Успешно опубликовали, теперь подтверждаем ОРИГИНАЛ
							c.baseConsumer.Logger.Info("Successfully published to final DLX. Acking original message",
								"consumer_tag", c.baseConsumer.config.ConsumerTag,
								"delivery_tag", delivery.DeliveryTag)
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

	// Теперь мы ждем либо отмены внешнего контекста, либо закрытия соединения.
	// Это решает deadlock.
	select {
	case <-ctx.Done():
		c.baseConsumer.Logger.Info("Context cancelled. Shutting down consumer.",
			"consumer_tag", c.baseConsumer.config.ConsumerTag)

		// Это штатное завершение. Мы получили сигнал, что пора выходить.
		// Внутренняя горутина тоже увидит ctx.Done() и завершится.
		// Мы возвращаем nil, потому что это не ошибка, а graceful shutdown.
		return nil

	case err := <-notifyClose:
		// Соединение было закрыто брокером или другим компонентом.
		// Это, как правило, ошибка, которую нужно обработать.
		c.baseConsumer.Logger.Error(err, "Connection closed for consumer.",
			"consumer_tag", c.baseConsumer.config.ConsumerTag)
		return err // Возвращаем ошибку от RabbitMQ
	}
}

// Close закрывает соединение потребителя
func (c *DistributingConsumer) Close() error {
	c.baseConsumer.Logger.Info("Closing consumer")

	return c.baseConsumer.Close()
}
