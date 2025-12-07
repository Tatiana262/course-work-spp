package internal

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	// "parser-project/internal/adapters/filestorage"
	"kufar-parser-service/internal/adapters/kufarfetcher"
	logger_adapter "kufar-parser-service/internal/adapters/logger"
	postgres_adapter "kufar-parser-service/internal/adapters/postgres"
	rabbitmq_adapter "kufar-parser-service/internal/adapters/rabbitmq"
	"kufar-parser-service/internal/configs"
	"kufar-parser-service/internal/constants"
	"kufar-parser-service/internal/core/port"
	"kufar-parser-service/internal/core/usecase"
	fluentlogger "real-estate-system/pkg/fluent_logger"
	"real-estate-system/pkg/postgres"
	"real-estate-system/pkg/rabbitmq/rabbitmq_common"
	"real-estate-system/pkg/rabbitmq/rabbitmq_consumer"
	"real-estate-system/pkg/rabbitmq/rabbitmq_producer"
	"sync"
	"syscall"

	// "time"

	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
)

// App – структура приложения
type App struct {
	config        *configs.AppConfig
	dbPool        *pgxpool.Pool
	connManager   *rabbitmq_common.ConnectionManager
	eventProducer *rabbitmq_producer.Publisher
	fluentClient  *fluent.Fluent
	logger        port.LoggerPort

	// Входящие порты (слушатели событий)
	linkEventsListener   port.EventListenerPort
	searchEventsListener port.EventListenerPort
}

// NewApp создает новый экземпляр приложения.
// Это "Composition Root", где все зависимости создаются и связываются.
func NewApp() (*App, error) {
	appConfig, err := configs.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading application configuration: %w", err)
	}

	// --- 1. ИНИЦИАЛИЗАЦИЯ ЛОГГЕРОВ ---
	var activeLoggers []port.LoggerPort

	slogCfg := logger_adapter.SlogConfig{
		Level:    parseLogLevel(appConfig.StdoutLogger.Level),
		IsJSON:   false, // текстовый формат
		UseColor: true,
	}
	stdoutLogger := logger_adapter.NewSlogAdapter(slogCfg)
	activeLoggers = append(activeLoggers, stdoutLogger)

	// Добавляем Fluent Bit логгер, если он включен в конфигурации
	// (предположим, что в appConfig.FluentBit есть поле Enabled bool)
	var fluentClient *fluent.Fluent
	if appConfig.FluentBit.Enabled {
		fluentClient, err = fluentlogger.NewClient(fluentlogger.Config{
			Host:      appConfig.FluentBit.Host,
			Port:      appConfig.FluentBit.Port,
			TagPrefix: appConfig.AppName, // Используем имя приложения как префикс
		})
		if err != nil {
			stdoutLogger.Error("Failed to create fluentbit client", err, nil)
			return nil, fmt.Errorf("failed to create fluentbit client: %w", err)
		}

		fluentAdapter, err := logger_adapter.NewFluentLoggerAdapter(fluentClient, parseLogLevel(appConfig.FluentBit.Level))
		if err != nil {
			stdoutLogger.Error("Failed to create fluentbit adapter", err, nil)
			fluentClient.Close()
			return nil, err
		}
		activeLoggers = append(activeLoggers, fluentAdapter)
	}

	// Создаем наш композитный логгер
	multiLogger, err := logger_adapter.NewMultiloggerAdapter(activeLoggers...)
	if err != nil {
		return nil, fmt.Errorf("failed to create multi-logger: %w", err)
	}

	// --- 2. СОЗДАЕМ БАЗОВЫЙ ЛОГГЕР ПРИЛОЖЕНИЯ С КОНТЕКСТОМ ---
	baseLogger := multiLogger.WithFields(port.Fields{
		"service_name": appConfig.AppName,
		// "service_version": "1.0.0",
	})

	appLogger := baseLogger.WithFields(port.Fields{"component": "app"})
	appLogger.Info("Logger system initialized", port.Fields{
		"active_loggers": len(activeLoggers), "fluent_enabled": appConfig.FluentBit.Enabled,
	})

	producerLogger := baseLogger.WithFields(port.Fields{"component": "rabbitmq_producer"})
	pkgLoggerBridge := rabbitmq_adapter.NewPkgLoggerBridge(producerLogger)

	connManagerLogger := baseLogger.WithFields(port.Fields{"component": "rabbitmq_conn_manager"})
	connManagerBridge := rabbitmq_adapter.NewPkgLoggerBridge(connManagerLogger)
	connManager, err := rabbitmq_common.GetManager(appConfig.RabbitMQ.URL, connManagerBridge)
	if err != nil {
		appLogger.Error("Failed to create connection manager", err, nil)
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}
	appLogger.Info("RabbitMQ Connection Manager initialized.", nil)

	// 1. Инициализация низкоуровневых зависимостей
	dbPool, err := postgres.NewClient(context.Background(), postgres.Config{DatabaseURL: appConfig.Database.URL})
	if err != nil {
		appLogger.Error("Failed to connect to PostgreSQL", err, nil)
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	appLogger.Info("Successfully connected to PostgreSQL pool!", nil)

	producerCfg := rabbitmq_producer.PublisherConfig{
		Config:                   rabbitmq_common.Config{URL: appConfig.RabbitMQ.URL},
		ExchangeName:             "parser_exchange",
		ExchangeType:             "direct",
		DurableExchange:          true,
		DeclareExchangeIfMissing: true,
		Logger:                   pkgLoggerBridge,
	}
	eventProducer, err := rabbitmq_producer.NewPublisher(producerCfg, connManager)
	if err != nil {
		appLogger.Error("Failed to create event producer", err, port.Fields{"url": appConfig.RabbitMQ.URL})
		dbPool.Close()
		return nil, fmt.Errorf("failed to create event producer: %w", err)
	}
	appLogger.Info("RabbitMQ Event Producer initialized.", nil)

	kufarAdapter, err := kufarfetcher.NewKufarFetcherAdapter(
		"https://api.kufar.by/search-api/v2/search/rendered-paginated",
	)
	if err != nil {
		appLogger.Error("Failed to create Kufar Fetcher Adapter", err, nil)
		eventProducer.Close()
		dbPool.Close()
		return nil, fmt.Errorf("failed to initialize kufar fetcher: %w", err)
	}
	appLogger.Info("Kufar Fetcher Adapter initialized.", nil)

	pgLastRunRepo, _ := postgres_adapter.NewPostgresLastRunRepository(dbPool)

	linkQueueAdapter, _ := rabbitmq_adapter.NewRabbitMQLinkQueueAdapter(eventProducer, constants.RoutingKeyLinkTasks)
	tasksResultsQueueAdapter, _ := rabbitmq_adapter.NewTaskReporterAdapter(eventProducer, constants.RoutingKeyTaskResults)
	processedPropertyQueueAdapter, _ := rabbitmq_adapter.NewRabbitMQProcessedPropertyQueueAdapter(eventProducer, constants.RoutingKeyProcessedProperties)

	appLogger.Info("All outgoing adapters initialized.", nil)

	// 3. ИНИЦИАЛИЗАЦИЯ USE CASES (ядра бизнес-логики)
	fetchKufarUseCase := usecase.NewFetchAndEnqueueLinksUseCase(kufarAdapter, linkQueueAdapter, pgLastRunRepo, "kufar")
	processLinkUseCase := usecase.NewProcessLinkUseCase(kufarAdapter, processedPropertyQueueAdapter)
	orchestrateParsingUseCase := usecase.NewOrchestrateParsingUseCase(fetchKufarUseCase, tasksResultsQueueAdapter)
	// savePropertyUseCase := usecase.NewSavePropertyUseCase(postgresStorageAdapter)
	appLogger.Info("All use cases initialized.", nil)

	// 4. ИНИЦИАЛИЗАЦИЯ ВХОДЯЩИХ АДАПТЕРОВ (те, которые ВЫЗЫВАЮТ наше ядро)
	linksConsumerCfg := rabbitmq_consumer.ConsumerConfig{
		Config:              rabbitmq_common.Config{URL: appConfig.RabbitMQ.URL},
		QueueName:           constants.QueueLinkTasks,
		RoutingKeyForBind:   constants.RoutingKeyLinkTasks,
		ExchangeNameForBind: "parser_exchange",
		PrefetchCount:       5,
		DurableQueue:        true,
		ConsumerTag:         "link-processor-adapter",
		DeclareQueue:        true,
		QueueArgs: amqp.Table{
			"x-max-priority": int32(4),
		},

		// --- НОВЫЕ НАСТРОЙКИ ---
		// 1. Включаем сам механизм
		EnableRetryMechanism: true,

		// 2. Настраиваем "сателлиты" для этой конкретной очереди.
		// Используем имя основной очереди как префикс для уникальности.
		RetryExchange: constants.QueueLinkTasks + "_retry_ex",
		RetryQueue:    constants.QueueLinkTasks + "_retry_wait_10s",
		RetryTTL:      10000, // 10 секунд в миллисекундах

		// 3. Указываем общую "свалку" для сообщений, исчерпавших все попытки.
		FinalDLXExchange:   constants.FinalDLXExchange,
		FinalDLQ:           constants.FinalDLQ,
		FinalDLQRoutingKey: constants.FinalDLQRoutingKey,

		// 4. Задаем количество ретраев (помимо первой попытки).
		MaxRetries: 3,
	}
	linkListener, err := rabbitmq_adapter.NewLinkConsumerAdapter(linksConsumerCfg, processLinkUseCase, baseLogger, connManager)
	if err != nil {
		appLogger.Error("Failed to initialize Link Events Listener", err, nil)
		eventProducer.Close()
		dbPool.Close()
		return nil, err
	}
	appLogger.Info("Link Events Listener initialized.", nil)

	tasksConsumerCfg := rabbitmq_consumer.ConsumerConfig{
		Config:              rabbitmq_common.Config{URL: appConfig.RabbitMQ.URL},
		QueueName:           constants.QueueSearchTasks,
		RoutingKeyForBind:   constants.RoutingKeySearchTasks,
		ExchangeNameForBind: "parser_exchange",
		PrefetchCount:       1,
		DurableQueue:        true,
		ConsumerTag:         "search-tasks-processor-adapter",
		DeclareQueue:        true,

		// --- НОВЫЕ НАСТРОЙКИ ---
		// 1. Включаем сам механизм
		EnableRetryMechanism: true,

		// 2. Настраиваем "сателлиты" для этой конкретной очереди.
		// Используем имя основной очереди как префикс для уникальности.
		RetryExchange: constants.QueueSearchTasks + "_retry_ex",
		RetryQueue:    constants.QueueSearchTasks + "_retry_wait_10s",
		RetryTTL:      10000, // 10 секунд в миллисекундах

		// 3. Указываем общую "свалку" для сообщений, исчерпавших все попытки.
		FinalDLXExchange:   constants.FinalDLXExchangeForSearchTasks,
		FinalDLQ:           constants.FinalDLQForSearchTasks,
		FinalDLQRoutingKey: constants.FinalDLQRoutingKeyForSearchTasks,

		// 4. Задаем количество ретраев (помимо первой попытки).
		MaxRetries: 3,
	}
	searchTasksListener, err := rabbitmq_adapter.NewTasksConsumerAdapter(tasksConsumerCfg, orchestrateParsingUseCase, baseLogger, connManager)
	if err != nil {
		appLogger.Error("Failed to initialize Search Events Listener", err, nil)
		eventProducer.Close()
		dbPool.Close()
		return nil, err
	}
	appLogger.Info("Search Events Listener initialized.", nil)

	// 5. Собираем приложение
	application := &App{
		config:        appConfig,
		dbPool:        dbPool,
		connManager:   connManager,
		fluentClient:  fluentClient,
		logger:        appLogger,
		eventProducer: eventProducer,
		// fetchKufarLinksUseCase:      fetchKufarUseCase, // Нужен для прямого вызова
		linkEventsListener:   linkListener,
		searchEventsListener: searchTasksListener,
	}

	return application, nil
}

// // StartKufarLinkFetcher запускает процесс сбора ссылок.
// func (a *App) StartKufarLinkFetcher(ctx context.Context) {
// 	log.Println("App: Initiating Kufar link fetching...")
// 	searches := constants.GetPredefinedSearches()

// 	for _, search := range searches {
// 		go func(crit domain.SearchCriteria, searchName string) {
// 			if err := a.fetchKufarLinksUseCase.Execute(ctx, crit); err != nil {
// 				log.Printf("App: Kufar link fetching for '%s' finished with error: %v", searchName, err)
// 			} else {
// 				log.Printf("App: Kufar link fetching for '%s' finished successfully.", searchName)
// 			}
// 		}(search.Criteria, search.Name)
// 	}
// }

// Run запускает все компоненты приложения и управляет их жизненным циклом.
func (a *App) Run() error {
	// Создаем единый контекст для всего приложения для управления graceful shutdown
	appCtx, cancelApp := context.WithCancel(context.Background())
	//defer cancelApp()

	// Используем WaitGroup для ожидания завершения всех фоновых задач
	var wg sync.WaitGroup

	defer func() {
		a.logger.Info("Shutdown sequence initiated...", nil)

		// Ждем завершения всех запущенных горутин (слушателей)
		a.logger.Info("Waiting for background processes to finish...", nil)
		wg.Wait()
		a.logger.Info("All background processes finished.", nil)

		// Теперь безопасно закрываем ресурсы
		if a.linkEventsListener != nil {
			if err := a.linkEventsListener.Close(); err != nil {
				a.logger.Error("Error closing links listener", err, nil)
			}
		}
		if a.searchEventsListener != nil {
			if err := a.searchEventsListener.Close(); err != nil {
				a.logger.Error("Error closing search tasks listener", err, nil)
			}
		}
		if a.eventProducer != nil {
			if err := a.eventProducer.Close(); err != nil {
				a.logger.Error("Error closing event producer", err, nil)
			}
		}

		if a.connManager != nil {
			if err := a.connManager.Close(); err != nil {
				a.logger.Error("Error closing RabbitMQ connection manager", err, nil)
			}
		}

		if a.dbPool != nil {
			a.dbPool.Close()
			a.logger.Info("PostgreSQL pool closed.", nil)
		}

		a.logger.Info("Application shut down gracefully.", nil)

		if a.fluentClient != nil {
			a.logger.Info("Closing Fluent Bit connection...", nil)
			if err := a.fluentClient.Close(); err != nil {
				log.Printf("App: Error closing fluent client: %v\n", err)
			}
		}

	}()

	a.logger.Info("Application is starting...", nil)

	// Запускаем сборщик ссылок, передавая ему главный контекст
	// a.StartKufarLinkFetcher(appCtx)

	consumerErrors := make(chan error, 1)

	// Функция-хелпер для запуска слушателей
	startListener := func(name string, listener port.EventListenerPort) {
		defer wg.Done()
		listenerLogger := a.logger.WithFields(port.Fields{"listener_name": name})
		listenerLogger.Info("Starting listener...", nil)

		if err := listener.Start(appCtx); err != nil {
			listenerLogger.Error("Listener stopped with an unexpected error", err, nil)
			consumerErrors <- fmt.Errorf("%s error: %w", name, err)
		} else {
			listenerLogger.Info("Listener stopped gracefully due to context cancellation.", nil)
		}
	}

	wg.Add(2)
	go startListener("Links Events Listener", a.linkEventsListener)
	go startListener("Search Events Listener", a.searchEventsListener)

	// Ожидание сигнала на завершение или ошибки от одного из компонентов
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	a.logger.Info("Application running. Waiting for signals or consumer error...", nil)
	select {
	case receivedSignal := <-quit:
		a.logger.Warn("Received signal, shutting down", port.Fields{"signal": receivedSignal.String()})
	case err := <-consumerErrors:
		a.logger.Error("A critical component failed, shutting down", err, nil)
	case <-appCtx.Done():
		a.logger.Warn("Context was cancelled unexpectedly, shutting down", nil)
	}

	// Инициируем graceful shutdown, отменяя главный контекст
	cancelApp()

	return nil
}

func parseLogLevel(levelStr string) slog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		// Возвращаем безопасное значение по умолчанию и логируем предупреждение
		log.Printf("Warning: Unknown log level '%s'. Defaulting to 'info'.", levelStr)
		return slog.LevelInfo
	}
}
