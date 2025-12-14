package internal

import (
	"context"
	"fmt"
	"log"
	"strings"

	// "log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	logger_adapter "storage-service/internal/adapters/logger"
	postgres_adapter "storage-service/internal/adapters/postgres"
	"storage-service/internal/adapters/rest"
	"storage-service/internal/configs"

	fluentlogger "real-estate-system/pkg/fluent_logger"
	"real-estate-system/pkg/postgres"
	"real-estate-system/pkg/rabbitmq/rabbitmq_common"
	"real-estate-system/pkg/rabbitmq/rabbitmq_consumer"
	"real-estate-system/pkg/rabbitmq/rabbitmq_producer"
	rabbitmq_adapter "storage-service/internal/adapters/rabbitmq"
	"storage-service/internal/constants"
	"storage-service/internal/core/port"
	"storage-service/internal/core/usecase"
	"sync"
	"syscall"

	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/jackc/pgx/v5/pgxpool"
)

// App – структура приложения
type App struct {
	config       *configs.AppConfig
	dbPool       *pgxpool.Pool
	apiServer    *rest.Server
	fluentClient *fluent.Fluent
	logger       port.LoggerPort

	processedPropEventsListener port.EventListenerPort
	tasksResultsProducer        *rabbitmq_producer.Publisher
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

	// 1. Инициализация низкоуровневых зависимостей
	dbPool, err := postgres.NewClient(context.Background(), postgres.Config{DatabaseURL: appConfig.Database.URL})
	if err != nil {
		appLogger.Error("Failed to connect to PostgreSQL", err, nil)
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	appLogger.Info("Successfully connected to PostgreSQL pool!", nil)

	postgresStorageAdapter, err := postgres_adapter.NewPostgresStorageAdapter(dbPool)
	if err != nil {
		appLogger.Error("Failed to create postgres storage adapter", err, nil)
		dbPool.Close()
		return nil, fmt.Errorf("failed to create postgres storage adapter: %w", err)
	}

	filterRepository, err := postgres_adapter.NewFilterRepository(dbPool)
	if err != nil {
		appLogger.Error("Failed to create postgres filter repository", err, nil)
		dbPool.Close()
		return nil, fmt.Errorf("failed to create postgres filter repository: %w", err)
	}

	appLogger.Info("Postgres storage adapters initialized.", nil)

	producerLogger := baseLogger.WithFields(port.Fields{"component": "rabbitmq_producer"})
	pkgLoggerBridge := rabbitmq_adapter.NewPkgLoggerBridge(producerLogger)

	connManagerLogger := baseLogger.WithFields(port.Fields{"component": "rabbitmq_conn_manager"})
	connManagerBridge := rabbitmq_adapter.NewPkgLoggerBridge(connManagerLogger)
	connManager, err := rabbitmq_common.GetManager(appConfig.RabbitMQ.URL, connManagerBridge)
	if err != nil {
		appLogger.Error("Failed to create connection manager", err, nil)
		dbPool.Close()
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}
	appLogger.Info("RabbitMQ Connection Manager initialized.", nil)

	producerCfg := rabbitmq_producer.PublisherConfig{
		Config:                   rabbitmq_common.Config{URL: appConfig.RabbitMQ.URL},
		ExchangeName:             "parser_exchange",
		ExchangeType:             "direct",
		DurableExchange:          true,
		DeclareExchangeIfMissing: true,

		Logger: pkgLoggerBridge,
	}
	eventProducer, err := rabbitmq_producer.NewPublisher(producerCfg, connManager)
	if err != nil {
		appLogger.Error("Failed to create event producer", err, nil)
		dbPool.Close()
		return nil, fmt.Errorf("failed to create event producer: %w", err)
	}
	appLogger.Info("RabbitMQ Event Producer initialized.", nil)

	tasksResultsQueueAdapter, _ := rabbitmq_adapter.NewTaskReporterAdapter(eventProducer, constants.RoutingKeyTaskResults)
	appLogger.Info("All outgoing adapters initialized.", nil)

	// ИНИЦИАЛИЗАЦИЯ USE CASES (ядра бизнес-логики)
	savePropertyUseCase := usecase.NewSavePropertyUseCase(postgresStorageAdapter, tasksResultsQueueAdapter)
	getActiveObjectsUseCase := usecase.NewGetActiveObjectsUseCase(postgresStorageAdapter)
	getArchivedObjectsUseCase := usecase.NewGetArchivedObjectsUseCase(postgresStorageAdapter)
	getObjectByIDUseCase := usecase.NewGetObjectsByIDUseCase(postgresStorageAdapter)

	findObjectsUseCase := usecase.NewFindObjectsUseCase(postgresStorageAdapter)
	getObjectDetailsUseCase := usecase.NewGetObjectDetailsUseCase(postgresStorageAdapter)
	getBestObjectsByMasterIDsUseCase := usecase.NewGetBestObjectsByMasterIDsUseCase(postgresStorageAdapter)

	getFilterOptionsUseCase := usecase.NewGetFilterOptionsUseCase(filterRepository)
	getDictionariesUseCase := usecase.NewGetDictionariesUseCase(filterRepository)

	appLogger.Info("All use cases initialized.", nil)

	// 4. ИНИЦИАЛИЗАЦИЯ ВХОДЯЩИХ АДАПТЕРОВ (те, которые ВЫЗЫВАЮТ наше ядро)
	processedConsumerCfg := rabbitmq_consumer.ConsumerConfig{
		Config:              rabbitmq_common.Config{URL: appConfig.RabbitMQ.URL},
		QueueName:           constants.QueueProcessedProperties,
		DurableQueue:        true,
		ExchangeNameForBind: "parser_exchange",
		RoutingKeyForBind:   constants.RoutingKeyProcessedProperties,
		PrefetchCount:       1,
		ConsumerTag:         "property-saver-adapter",
		DeclareQueue:        true,

		// 1. Включаем механизм
		EnableRetryMechanism: true,

		// 2. Настраиваем уникальные "сателлиты" для этой очереди
		RetryExchange: constants.QueueProcessedProperties + "_retry_ex",
		RetryQueue:    constants.QueueProcessedProperties + "_retry_wait_10s",
		RetryTTL:      10000, // 10 секунд в миллисекундах

		// 3. Используем тот же самый общий финальный DLQ
		FinalDLXExchange:   constants.FinalDLXExchange,
		FinalDLQ:           constants.FinalDLQ,
		FinalDLQRoutingKey: constants.FinalDLQRoutingKey,

		// 4. Количество ретраев может быть другим, но 3 - хорошее начало.
		MaxRetries: 3,
	}
	processedPropListener, err := rabbitmq_adapter.NewProcessedPropertyConsumerAdapter(processedConsumerCfg, savePropertyUseCase, baseLogger, connManager)
	if err != nil {
		appLogger.Error("Failed to create Processed Property listener", err, nil)
		dbPool.Close()
		return nil, err
	}
	appLogger.Info("Processed Property Events Listener initialized.", nil)

	// REST API Server
	apiActualizationHandlers := rest.NewActualizationHandlers(getActiveObjectsUseCase, getArchivedObjectsUseCase, getObjectByIDUseCase)
	apiGetInfoHandlers := rest.NewGetInfoHandler(findObjectsUseCase, getObjectDetailsUseCase, getBestObjectsByMasterIDsUseCase)
	filtersHandlers := rest.NewFilterHandler(getFilterOptionsUseCase, getDictionariesUseCase)

	apiServer := rest.NewServer(appConfig.Rest.PORT, apiActualizationHandlers, apiGetInfoHandlers, filtersHandlers, baseLogger)
	appLogger.Info("REST API server configured.", nil)

	// 5. Собираем приложение
	application := &App{
		config:                      appConfig,
		dbPool:                      dbPool,
		apiServer:                   apiServer,
		processedPropEventsListener: processedPropListener,
		tasksResultsProducer:        eventProducer,

		fluentClient: fluentClient,
		logger:       appLogger,
	}

	return application, nil
}

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

		if a.apiServer != nil {
			if err := a.apiServer.Stop(context.Background()); err != nil {
				a.logger.Error("Error during API server shutdown", err, nil)
			}
		}

		// Теперь безопасно закрываем ресурсы
		if a.processedPropEventsListener != nil {
			if err := a.processedPropEventsListener.Close(); err != nil {
				a.logger.Error("Error closing processed properties listener", err, nil)
			}
		}

		if a.tasksResultsProducer != nil {
			if err := a.tasksResultsProducer.Close(); err != nil {
				a.logger.Error("Error closing event producer", err, nil)
			}
		}

		if a.dbPool != nil {
			a.dbPool.Close()
			a.logger.Info("PostgreSQL pool closed.", nil)
		}

		a.logger.Info("Application shut down gracefully.", nil)

		if a.fluentClient != nil {
			if err := a.fluentClient.Close(); err != nil {
				// Логируем в stdout, так как fluent может быть уже недоступен
				fmt.Printf("ERROR: Error closing fluent client: %v\n", err)
			}
		}

	}()

	a.logger.Info("Application is starting...", nil)

	errorsCh := make(chan error, 1)

	// Функция-хелпер для запуска слушателей
	startListener := func(name string, listener port.EventListenerPort) {
		defer wg.Done()
		listenerLogger := a.logger.WithFields(port.Fields{"listener_name": name})
		listenerLogger.Info("Starting listener...", nil)

		if err := listener.Start(appCtx); err != nil {
			listenerLogger.Error("Listener stopped with an unexpected error", err, nil)
			errorsCh <- fmt.Errorf("%s error: %w", name, err)
		} else {
			listenerLogger.Info("Listener stopped gracefully due to context cancellation.", nil)
		}
	}

	wg.Add(1)
	go startListener("Processed Property Events Listener", a.processedPropEventsListener)

	go func() {
		a.logger.Info("Starting HTTP server...", port.Fields{"port": a.config.Rest.PORT})
		if err := a.apiServer.Start(); err != nil && err != http.ErrServerClosed {
			errorsCh <- fmt.Errorf("failed to start HTTP server: %w", err)
		}
	}()

	// Ожидание сигнала на завершение или ошибки от одного из компонентов
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	a.logger.Info("Application running. Waiting for signals or server error...", nil)
	select {
	case receivedSignal := <-quit:
		a.logger.Warn("Received OS signal, shutting down...", port.Fields{"signal": receivedSignal.String()})
	case err := <-errorsCh:
		a.logger.Error("A critical component failed, shutting down", err, nil)
	case <-appCtx.Done():
		a.logger.Warn("Context was cancelled unexpectedly, shutting down...", nil)
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