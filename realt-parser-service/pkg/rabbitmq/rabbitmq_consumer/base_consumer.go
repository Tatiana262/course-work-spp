package rabbitmq_consumer

import (
	"fmt"
	"realt-parser-service/pkg/rabbitmq/rabbitmq_common"
	"realt-parser-service/pkg/rabbitmq/rabbitmq_producer"
	"log"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// baseConsumer содержит общую логику подключения
type baseConsumer struct {
	config     ConsumerConfig
    connection    *amqp.Connection
    channel *amqp.Channel
    actualQueueName string		// для хранения имени очереди, если оно генерируется сервером
	finalDlxPublisher *rabbitmq_producer.Publisher 
	wg           	sync.WaitGroup // для graceful shutdown
}

// ConsumerConfig конфигурация для потребителя
type ConsumerConfig struct {
	rabbitmq_common.Config
	// Настройки очереди
	QueueName             string // Имя очереди для потребления (если пусто, имя будет сгенерировано сервером)
	DeclareQueue          bool   // Пытаться ли объявить очередь
	DurableQueue          bool
	ExclusiveQueue        bool
	AutoDeleteQueue       bool
	QueueArgs             amqp.Table // Дополнительные аргументы для очереди (например, x-message-ttl, x-dead-letter-exchange)
	// Настройки обменника (если нужно объявлять или привязываться к нему)
	ExchangeNameForBind   string // Имя обменника для привязки очереди (если пусто, привязка не выполняется)
	DeclareExchangeForBind bool  // Пытаться ли объявить этот обменник
	ExchangeTypeForBind   string // Тип этого обменника
	DurableExchangeForBind bool
	ExchangeArgsForBind   amqp.Table // Аргументы для обменника, если объявляем его
	// Настройки привязки
	RoutingKeyForBind     string     // Ключ маршрутизации для привязки
	BindingArgs           amqp.Table // Дополнительные аргументы для привязки
	// Настройки QoS
	PrefetchCount         int // 0 или меньше - без ограничений
	PrefetchSize          int // 0 - без ограничений
	QosGlobal             bool
	// Настройки потребителя
	ConsumerTag           string // Тег потребителя (если пустой, генерируется RabbitMQ)
	ExclusiveConsumer     bool

	EnableRetryMechanism bool   // флаг для включения механизма ретраев
	RetryExchange        string // Имя retry-обменника
	RetryQueue           string // Имя wait-очереди
	RetryTTL             int    // TTL для wait-очереди в миллисекундах
	FinalDLXExchange     string // Имя финального DLX
	FinalDLQ             string // Имя финальной DLQ
	FinalDLQRoutingKey   string // Ключ для привязки финальной DLQ
	MaxRetries           int    // Максимальное количество попыток (ретраев)
}


func newBaseConsumer(cfg ConsumerConfig) (*baseConsumer, error) {
    if err := cfg.Validate(); err != nil { // Валидация общей части
		return nil, fmt.Errorf("Base Consumer: invalid base config: %w", err)
	}
	// Валидация специфичная для ConsumerConfig
	if !cfg.DeclareQueue && cfg.QueueName == "" {
		return nil, fmt.Errorf("Base Consumer: queue name is required if DeclareQueue is false")
	}
	if cfg.ExchangeNameForBind != "" && cfg.ExchangeTypeForBind == "" && cfg.DeclareExchangeForBind {
		return nil, fmt.Errorf("Base Consumer: exchange type is required if declaring an exchange for binding")
	}
	
	c := &baseConsumer{
		config:  cfg,
	}

	if err := c.connectAndSetup(); err != nil {
		return nil, fmt.Errorf("Base Consumer: initial connection and setup failed: %w", err)
	}

	if cfg.EnableRetryMechanism {
		dlxPublisher, err := rabbitmq_producer.NewPublisher(rabbitmq_producer.PublisherConfig{
			Config: rabbitmq_common.Config{URL: cfg.URL},
			ExchangeName:             cfg.FinalDLXExchange,
			DeclareExchangeIfMissing: false,
		})
		if err != nil {
			_ = c.Close() 
			return nil, fmt.Errorf("Base Consumer: failed to create final DLX publisher: %w", err)
		}
		c.finalDlxPublisher = dlxPublisher
	}

	return c, nil
}


// connectAndSetup устанавливает соединение, канал и настраивает сущности RabbitMQ
func (c *baseConsumer) connectAndSetup() error {
	log.Printf("Base Consumer: Attempting to connect to RabbitMQ at %s\n", c.config.URL)
	conn, err := amqp.Dial(c.config.URL)
	if err != nil {
		return fmt.Errorf("Base Consumer: failed to dial RabbitMQ: %w", err)
	}
	c.connection = conn

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to open a channel: %w", err)
	}
	c.channel = ch
	log.Println("Base Consumer: Channel opened.")

	// 1. Настройка QoS
	if c.config.PrefetchCount > 0 || c.config.PrefetchSize > 0 {
		log.Printf("Base Consumer: Setting QoS (PrefetchCount: %d, PrefetchSize: %d, Global: %v)\n",
			c.config.PrefetchCount, c.config.PrefetchSize, c.config.QosGlobal)
		err = c.channel.Qos(
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
		// Указываем, что мертвые сообщения из основной очереди должны идти в retry-exchange
		c.config.QueueArgs["x-dead-letter-exchange"] = c.config.RetryExchange
	}

	c.actualQueueName = c.config.QueueName
	// 2. Объявление очереди
	if c.config.DeclareQueue {
		log.Printf("Base Consumer: Declaring queue '%s' (durable: %v, exclusive: %v, autoDelete: %v)\n",
			c.config.QueueName, c.config.DurableQueue, c.config.ExclusiveQueue, c.config.AutoDeleteQueue)
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


	// 3. Объявление обменника
	if c.config.DeclareExchangeForBind {
		log.Printf("Base Consumer: Declaring exchange '%s' for binding (type: %s, durable: %v)\n",
			c.config.ExchangeNameForBind, c.config.ExchangeTypeForBind, c.config.DurableExchangeForBind)
		err = c.channel.ExchangeDeclare(
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

	// 4. Привязка очереди к обменнику
	if c.config.ExchangeNameForBind != "" {
		log.Printf("Base Consumer: Binding queue '%s' to exchange '%s' with routing key '%s'\n",
			c.actualQueueName, c.config.ExchangeNameForBind, c.config.RoutingKeyForBind)
		err = c.channel.QueueBind(
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


	// 5. объявление инфраструктуры ретраев
	if c.config.EnableRetryMechanism {
		log.Println("Base Consumer: Setting up isolated retry mechanism...")

		// объявляем финальный DLX и DLQ (куда попадают сообщения после всех ретраев)
		log.Printf("Base Consumer: Declaring final DLX '%s'", c.config.FinalDLXExchange)
		err := c.channel.ExchangeDeclare(c.config.FinalDLXExchange, "direct", true, false, false, false, nil)
		if err != nil { return fmt.Errorf("failed to declare final DLX: %w", err) }

		log.Printf("Base Consumer: Declaring final DLQ '%s'", c.config.FinalDLQ)
		_, err = c.channel.QueueDeclare(c.config.FinalDLQ, true, false, false, false, nil)
		if err != nil { return fmt.Errorf("failed to declare final DLQ: %w", err) }

		// Привязываем финальную DLQ к финальному DLX
		log.Printf("Base Consumer: Binding final DLQ '%s' to DLX '%s' with key '%s'", c.config.FinalDLQ, c.config.FinalDLXExchange, c.config.FinalDLQRoutingKey)
		err = c.channel.QueueBind(c.config.FinalDLQ, c.config.FinalDLQRoutingKey, c.config.FinalDLXExchange, false, nil)
		if err != nil { return fmt.Errorf("failed to bind final DLQ: %w", err) }

		// Объявляем обменник для ретраев
		log.Printf("Base Consumer: Declaring retry exchange '%s'", c.config.RetryExchange)
		err = c.channel.ExchangeDeclare(c.config.RetryExchange, "fanout", true, false, false, false, nil)
		if err != nil { return fmt.Errorf("failed to declare retry exchange: %w", err) }

		// Объявляем очередь ожидания с TTL, которая возвращает сообщения в основной обменник
		log.Printf("Base Consumer: Declaring retry-wait queue '%s' with TTL %dms", c.config.RetryQueue, c.config.RetryTTL)
		_, err = c.channel.QueueDeclare(
			c.config.RetryQueue,
			true,  // durable
			false, // autoDelete
			false, // exclusive
			false, // noWait
			amqp.Table{
				"x-message-ttl":             int32(c.config.RetryTTL),
				"x-dead-letter-exchange":    c.config.ExchangeNameForBind, // Возвращаем в основной обменник
				"x-dead-letter-routing-key": c.config.RoutingKeyForBind,   // с основным ключом
			},
		)
		if err != nil { return fmt.Errorf("failed to declare retry-wait queue: %w", err) }

		// Привязываем очередь ожидания к retry-обменнику
		err = c.channel.QueueBind(c.config.RetryQueue, "", c.config.RetryExchange, false, nil)
		if err != nil { return fmt.Errorf("failed to bind retry-wait queue: %w", err) }
	}


	log.Printf("Base Consumer: Setup complete for queue '%s'.\n", c.actualQueueName)
	return nil
}


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

	// x-death - это массив смертей
	for _, death := range deaths {
		if tbl, ok := death.(amqp.Table); ok {
			// сколько раз сообщение умирало в основной очереди
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

	log.Println("Base Consumer: Waiting for message handlers to finish...")
	c.wg.Wait()
	log.Println("Base Consumer: All message handlers finished.")

	var firstErr error

	if c.finalDlxPublisher != nil {
		if err := c.finalDlxPublisher.Close(); err != nil {
			log.Printf("Base Consumer: Error closing final DLX publisher: %v\n", err)
			firstErr = err
		}
	}

	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			log.Printf("Base Consumer: Error closing channel: %v\n", err)
			firstErr = err
		}
		c.channel = nil
	}
	if c.connection != nil {
		if err := c.connection.Close(); err != nil {
			log.Printf("Base Consumer: Error closing connection: %v\n", err)
			if firstErr == nil {
				firstErr = err
			}
		}
		c.connection = nil
	}
	log.Println("Base Consumer: Closed.")
	return firstErr
}