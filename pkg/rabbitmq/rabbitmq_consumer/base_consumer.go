package rabbitmq_consumer

import (
	"fmt"
	"real-estate-system/pkg/rabbitmq/rabbitmq_common"
	"real-estate-system/pkg/rabbitmq/rabbitmq_producer"

	// "log"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// baseConsumer содержит общую логику подключения, канала, QoS и т.д.
type baseConsumer struct {
	config            ConsumerConfig
	connection        *amqp.Connection
	channel           *amqp.Channel
	actualQueueName   string                       // Для хранения имени очереди, особенно если оно генерируется сервером
	finalDlxPublisher *rabbitmq_producer.Publisher // <<-- ПЕРЕНОСИМ СЮДА
	wg                sync.WaitGroup               // Нужен для graceful shutdown

	Logger rabbitmq_common.Logger
}

// ConsumerConfig конфигурация для потребителя
type ConsumerConfig struct {
	rabbitmq_common.Config
	// Настройки очереди
	QueueName       string // Имя очереди для потребления (если пусто, имя будет сгенерировано сервером)
	DeclareQueue    bool   // Пытаться ли объявить очередь
	DurableQueue    bool
	ExclusiveQueue  bool
	AutoDeleteQueue bool
	QueueArgs       amqp.Table // Дополнительные аргументы для очереди (например, x-message-ttl, x-dead-letter-exchange)
	// Настройки обменника (если нужно объявлять или привязываться к нему)
	ExchangeNameForBind    string // Имя обменника для привязки очереди (если пусто, привязка не выполняется)
	DeclareExchangeForBind bool   // Пытаться ли объявить этот обменник
	ExchangeTypeForBind    string // Тип этого обменника
	DurableExchangeForBind bool
	ExchangeArgsForBind    amqp.Table // Аргументы для обменника, если объявляем его
	// Настройки привязки
	RoutingKeyForBind string     // Ключ маршрутизации для привязки
	BindingArgs       amqp.Table // Дополнительные аргументы для привязки
	// Настройки QoS
	PrefetchCount int // 0 или меньше - без ограничений
	PrefetchSize  int // 0 - без ограничений
	QosGlobal     bool
	// Настройки потребителя
	ConsumerTag       string // Тег потребителя (если пустой, генерируется RabbitMQ)
	ExclusiveConsumer bool

	// --- НОВЫЕ ПОЛЯ ДЛЯ ИЗОЛИРОВАННЫХ РЕТРАЕВ ---
	EnableRetryMechanism bool   // Главный флаг для включения
	RetryExchange        string // Имя retry-обменника (e.g., "my_queue_retry_ex")
	RetryQueue           string // Имя wait-очереди (e.g., "my_queue_retry_wait_10s")
	RetryTTL             int    // TTL для wait-очереди в миллисекундах
	FinalDLXExchange     string // Имя финального DLX (может быть общим)
	FinalDLQ             string // Имя финальной DLQ (может быть общим)
	FinalDLQRoutingKey   string // Ключ для привязки финальной DLQ
	MaxRetries           int    // Максимальное количество попыток (1 + N ретраев)

	Logger rabbitmq_common.Logger
}

// newBaseConsumer инкапсулирует всю логику из вашего connectAndSetup
func newBaseConsumer(cfg ConsumerConfig, connManager *rabbitmq_common.ConnectionManager) (*baseConsumer, error) {

	logger := cfg.Logger
	if logger == nil {
		logger = rabbitmq_common.NewNoopLogger()
	}

	if err := cfg.Validate(); err != nil { // Валидация общей части
		return nil, fmt.Errorf("base Consumer: invalid base config: %w", err)
	}
	// Валидация специфичная для ConsumerConfig
	if !cfg.DeclareQueue && cfg.QueueName == "" {
		return nil, fmt.Errorf("base Consumer: queue name is required if DeclareQueue is false")
	}
	if cfg.ExchangeNameForBind != "" && cfg.ExchangeTypeForBind == "" && cfg.DeclareExchangeForBind {
		return nil, fmt.Errorf("base Consumer: exchange type is required if declaring an exchange for binding")
	}

	c := &baseConsumer{
		config: cfg,
		Logger: logger,
	}

	conn, ch, err := connManager.GetChannel()
	if err != nil {
		return nil, fmt.Errorf("base Consumer: failed to get channel from manager: %w", err)
	}
	c.connection = conn // Сохраняем ссылку для NotifyClose
	c.channel = ch
	c.Logger.Info("Channel obtained from ConnectionManager")

	if err := c.connectAndSetup(); err != nil {
		return nil, fmt.Errorf("base Consumer: initial connection and setup failed: %w", err)
	}

	if cfg.EnableRetryMechanism {
		dlxPublisher, err := rabbitmq_producer.NewPublisher(rabbitmq_producer.PublisherConfig{
			Config:                   rabbitmq_common.Config{URL: cfg.URL},
			ExchangeName:             cfg.FinalDLXExchange,
			DeclareExchangeIfMissing: false, // Уже объявлен в connectAndSetup
		}, connManager)
		if err != nil {
			_ = c.Close() // Важно почистить ресурсы, если что-то пошло не так
			return nil, fmt.Errorf("base Consumer: failed to create final DLX publisher: %w", err)
		}
		c.finalDlxPublisher = dlxPublisher
	}

	return c, nil
}

// connectAndSetup устанавливает соединение, канал и настраивает сущности RabbitMQ
func (c *baseConsumer) connectAndSetup() error {

	// c.Logger.Info("Attempting to connect to RabbitMQ", "url", c.config.URL)
	// conn, err := amqp.Dial(c.config.URL)
	// if err != nil {
	// 	return fmt.Errorf("base Consumer: failed to dial RabbitMQ: %w", err)
	// }
	// c.connection = conn

	// ch, err := conn.Channel()
	// if err != nil {
	// 	_ = conn.Close()
	// 	return fmt.Errorf("failed to open a channel: %w", err)
	// }
	// c.channel = ch
	// c.Logger.Info("Channel opened")

	// 1. Настройка QoS (должна быть до Consume)
	if c.config.PrefetchCount > 0 || c.config.PrefetchSize > 0 {
		c.Logger.Info("Setting QoS",
			"prefetch_count", c.config.PrefetchCount,
			"prefetch_size", c.config.PrefetchSize,
			"global", c.config.QosGlobal,
		)

		err := c.channel.Qos(
			c.config.PrefetchCount,
			c.config.PrefetchSize,
			c.config.QosGlobal,
		)
		if err != nil {
			_ = c.channel.Close()
			_ = c.connection.Close()
			return fmt.Errorf("failed to set QoS: %w", err)
		}
	}

	if c.config.EnableRetryMechanism {
		if c.config.QueueArgs == nil {
			c.config.QueueArgs = amqp.Table{}
		}
		// Указываем, что "мертвые" сообщения из основной очереди должны идти в retry-exchange
		c.config.QueueArgs["x-dead-letter-exchange"] = c.config.RetryExchange
	}

	c.actualQueueName = c.config.QueueName
	// 2. Объявление очереди (если нужно)
	if c.config.DeclareQueue {

		c.Logger.Info("Declaring queue",
			"name", c.config.QueueName,
			"durable", c.config.DurableQueue,
			"exclusive", c.config.ExclusiveQueue,
			"autoDelete", c.config.AutoDeleteQueue,
		)
		q, declareErr := c.channel.QueueDeclare(
			c.config.QueueName,       // name
			c.config.DurableQueue,    // durable
			c.config.AutoDeleteQueue, // delete when unused
			c.config.ExclusiveQueue,  // exclusive
			false,                    // no-wait
			c.config.QueueArgs,       // arguments
		)
		if declareErr != nil {
			_ = c.channel.Close()
			_ = c.connection.Close()
			return fmt.Errorf("failed to declare queue '%s': %w", c.config.QueueName, declareErr)
		}
		c.actualQueueName = q.Name // Используем имя, возвращенное сервером
	}

	// 4. Объявление обменника (если нужно для привязки)
	if c.config.DeclareExchangeForBind {

		c.Logger.Info("Declaring exchange",
			"name", c.config.ExchangeNameForBind,
			"type", c.config.ExchangeTypeForBind,
			"durable", c.config.DurableExchangeForBind,
		)
		err := c.channel.ExchangeDeclare(
			c.config.ExchangeNameForBind,
			c.config.ExchangeTypeForBind,
			c.config.DurableExchangeForBind,
			false, // auto-deleted
			false, // internal
			false, // no-wait
			c.config.ExchangeArgsForBind,
		)
		if err != nil {
			_ = c.channel.Close()
			_ = c.connection.Close()
			return fmt.Errorf("failed to declare exchange '%s' for binding: %w", c.config.ExchangeNameForBind, err)
		}
	}

	// 5. Привязка очереди к обменнику (если нужно)
	if c.config.ExchangeNameForBind != "" {

		c.Logger.Info("Binding queue to exchange",
			"queue_name", c.actualQueueName,
			"exchange_name", c.config.ExchangeNameForBind,
			"routing_key", c.config.RoutingKeyForBind,
		)
		err := c.channel.QueueBind(
			c.actualQueueName,
			c.config.RoutingKeyForBind,
			c.config.ExchangeNameForBind,
			false, // noWait
			c.config.BindingArgs,
		)
		if err != nil {
			_ = c.channel.Close()
			_ = c.connection.Close()
			return fmt.Errorf("failed to bind queue '%s' to exchange '%s': %w", c.actualQueueName, c.config.ExchangeNameForBind, err)
		}
	}

	// --- НОВЫЙ БЛОК: ОБЪЯВЛЕНИЕ ИНФРАСТРУКТУРЫ РЕТРАЕВ ---
	if c.config.EnableRetryMechanism {

		c.Logger.Info("Setting up isolated retry mechanism...")

		// A. Объявляем финальный DLX и DLQ (куда попадают сообщения после всех ретраев)
		c.Logger.Info("Declaring final DLX", "name", c.config.FinalDLXExchange)

		err := c.channel.ExchangeDeclare(c.config.FinalDLXExchange, "direct", true, false, false, false, nil)
		if err != nil {
			return fmt.Errorf("failed to declare final DLX: %w", err)
		}

		c.Logger.Info("Declaring final DLQ", "name", c.config.FinalDLQ)
		_, err = c.channel.QueueDeclare(c.config.FinalDLQ, true, false, false, false, nil)
		if err != nil {
			return fmt.Errorf("failed to declare final DLQ: %w", err)
		}

		// Привязываем финальную DLQ к финальному DLX
		c.Logger.Info("Binding final DLQ to DLX",
			"dlq_name", c.config.FinalDLQ,
			"dlx_name", c.config.FinalDLXExchange,
			"routing_key", c.config.FinalDLQRoutingKey,
		)
		err = c.channel.QueueBind(c.config.FinalDLQ, c.config.FinalDLQRoutingKey, c.config.FinalDLXExchange, false, nil)
		if err != nil {
			return fmt.Errorf("failed to bind final DLQ: %w", err)
		}

		// B. Объявляем обменник для ретраев (fanout - самый простой и надежный тип)
		c.Logger.Info("Declaring retry exchange", "name", c.config.RetryExchange)
		err = c.channel.ExchangeDeclare(c.config.RetryExchange, "fanout", true, false, false, false, nil)
		if err != nil {
			return fmt.Errorf("failed to declare retry exchange: %w", err)
		}

		// C. Объявляем очередь ожидания с TTL, которая возвращает сообщения в основной обменник
		c.Logger.Info("Declaring retry-wait queue with TTL",
			"name", c.config.RetryQueue,
			"ttl", c.config.RetryTTL,
		)
		_, err = c.channel.QueueDeclare(
			c.config.RetryQueue,
			true,  // durable
			false, // autoDelete
			false, // exclusive
			false, // noWait
			amqp.Table{
				"x-message-ttl":             int32(c.config.RetryTTL),
				"x-dead-letter-exchange":    c.config.ExchangeNameForBind, // Возвращаем в ОСНОВНОЙ обменник
				"x-dead-letter-routing-key": c.config.RoutingKeyForBind,   // С ОСНОВНЫМ ключом
			},
		)
		if err != nil {
			return fmt.Errorf("failed to declare retry-wait queue: %w", err)
		}

		// D. Привязываем очередь ожидания к retry-обменнику
		err = c.channel.QueueBind(c.config.RetryQueue, "", c.config.RetryExchange, false, nil)
		if err != nil {
			return fmt.Errorf("failed to bind retry-wait queue: %w", err)
		}
	}

	c.Logger.Info("Setup complete", "queue", c.actualQueueName)
	return nil
}

// getDeathCount - ОБНОВЛЕННАЯ ВЕРСИЯ, чтобы корректно работать с x-death
func (c *baseConsumer) getDeathCount(d amqp.Delivery, queueName string) int64 {
	if d.Headers == nil {
		return 0
	}
	xDeath, ok := d.Headers["x-death"]
	if !ok {
		return 0
	}
	deaths, ok := xDeath.([]interface{})
	if !ok {
		return 0
	}

	// x-death - это массив "смертей". Последняя "смерть" была в retry-очереди,
	// а нас интересует, сколько раз сообщение "умирало" в основной очереди.
	for _, death := range deaths {
		if tbl, ok := death.(amqp.Table); ok {
			// Ищем запись, где причиной смерти была наша основная очередь
			if queue, ok := tbl["queue"].(string); ok && queue == queueName {
				if count, ok := tbl["count"].(int64); ok {
					return count
				}
			}
		}
	}
	return 0
}

// Close закрывает соединение потребителя
func (c *baseConsumer) Close() error {

	c.Logger.Info("Waiting for message handlers to finish...")
	c.wg.Wait()
	c.Logger.Info("All message handlers finished")

	var firstErr error

	// ЗАКРЫВАЕМ ИЗДАТЕЛЯ
	if c.finalDlxPublisher != nil {
		if err := c.finalDlxPublisher.Close(); err != nil {
			c.Logger.Error(err, "Error closing final DLX publisher")
			firstErr = err
		}
	}

	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			c.Logger.Error(err, "Error closing channel")
			firstErr = err
		}
		c.channel = nil
	}
	// if c.connection != nil {
	// 	if err := c.connection.Close(); err != nil {
	// 		c.Logger.Error(err, "Error closing connection")
	// 		if firstErr == nil {
	// 			firstErr = err
	// 		}
	// 	}
	// 	c.connection = nil
	// }

	c.Logger.Info("Consumer closed")
	return firstErr
}

// // 3. Объявление Dead Letter Exchange и Queue (если включено)
// if c.config.EnableDLQ {
// 	log.Printf("Consumer: Declaring Dead Letter Exchange '%s'\n", c.config.DLXName)
// 	err = c.channel.ExchangeDeclare(
// 		c.config.DLXName,
// 		"direct", // DLX обычно типа direct
// 		true,     // durable
// 		false,
// 		false,
// 		false,
// 		nil,
// 	)
// 	if err != nil {
// 		// ... (закрытие соединений и возврат ошибки) ...
// 		return fmt.Errorf("failed to declare DLX '%s': %w", c.config.DLXName, err)
// 	}

// 	dlqName := c.config.QueueName + "_dlq"
// 	dlqRoutingKey := fmt.Sprintf("%s.%s", c.config.DLQRoutingKeyPrefix, c.config.QueueName)

// 	log.Printf("Consumer: Declaring Dead Letter Queue '%s'\n", dlqName)
// 	_, err = c.channel.QueueDeclare(dlqName, true, false, false, false, nil)
// 	if err != nil {
// 		_ = c.channel.Close()
// 		_ = c.connection.Close()
// 		return fmt.Errorf("failed to declare DLQ '%s': %w", dlqName, err)
// 	}

// 	log.Printf("Consumer: Binding DLQ '%s' to DLX '%s' with key '%s'\n", dlqName, c.config.DLXName, dlqRoutingKey)
// 	err = c.channel.QueueBind(dlqName, dlqRoutingKey, c.config.DLXName, false, nil)
// 	if err != nil {
// 		_ = c.channel.Close()
// 		_ = c.connection.Close()
// 		return fmt.Errorf("failed to bind DLQ: %w", err)
// 	}
// }
