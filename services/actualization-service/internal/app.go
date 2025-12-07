package internal

import (
	logger_adapter "actualization-service/internal/adapters/logger"
	"actualization-service/internal/adapters/rest"
	"actualization-service/internal/adapters/storage_api_client"
	"actualization-service/internal/adapters/task_api_client"
	"actualization-service/internal/configs"
	"actualization-service/internal/constants"
	"actualization-service/internal/core/port"
	"actualization-service/internal/core/usecase"
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	fluentlogger "real-estate-system/pkg/fluent_logger"
	"real-estate-system/pkg/rabbitmq/rabbitmq_common"
	"real-estate-system/pkg/rabbitmq/rabbitmq_producer"
	"strings"
	"syscall"

	rabbitmq_adapter "actualization-service/internal/adapters/rabbitmq"

	"github.com/fluent/fluent-logger-golang/fluent"
)

type App struct {
	config    *configs.AppConfig
	apiServer *rest.Server

	eventProducer *rabbitmq_producer.Publisher
	logger        port.LoggerPort // <-- ДОБАВЛЯЕМ ЛОГГЕР В СТРУКТУРУ
	fluentClient  *fluent.Fluent  // <-- ОСТАВЛЯЕМ ДЛЯ КОРРЕКТНОГО ЗАКРЫТИЯ
}

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
	pkgLoggerBridge := rabbitmq_adapter.NewPkgLoggerBridge(producerLogger) // Используем мост

	connManagerLogger := baseLogger.WithFields(port.Fields{"component": "rabbitmq_conn_manager"})
	connManagerBridge := rabbitmq_adapter.NewPkgLoggerBridge(connManagerLogger)
	connManager, err := rabbitmq_common.GetManager(appConfig.RabbitMQ.URL, connManagerBridge)
	if err != nil {
		appLogger.Error("Failed to create connection manager", err, nil)
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}
	appLogger.Info("RabbitMQ Connection Manager initialized.", nil)

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
		return nil, fmt.Errorf("failed to create event producer: %w", err)
	}
	appLogger.Info("RabbitMQ Event Producer initialized.", nil)

	tasksQueueAdapter, _ := rabbitmq_adapter.NewRabbitMQLinkQueueAdapter(eventProducer)
	tasksResultsAdapter, _ := rabbitmq_adapter.NewTaskManagerPublisher(eventProducer, constants.RoutingKeyTasksResults)
	storageClient := storage_api_client.NewClient(appConfig.ApiClient.STORAGE_URL)
	userTasksClient := task_api_client.NewClient(appConfig.ApiClient.TASKS_SERVICE_URL)

	linksSearchQueueAdapter, _ := rabbitmq_adapter.NewRabbitMQLinksSearchQueueAdapter(eventProducer)

	// ИНИЦИАЛИЗАЦИЯ USE CASES (ядра бизнес-логики)
	actualizeActiveObjectsUseCase := usecase.NewActualizeActiveObjectsUseCase(storageClient, tasksQueueAdapter, userTasksClient, tasksResultsAdapter)
	actualizeArchivedObjectsUseCase := usecase.NewActualizeArchivedObjectsUseCase(storageClient, tasksQueueAdapter, userTasksClient, tasksResultsAdapter)
	actualizeObjectByIdUseCase := usecase.NewActualizeObjectByIdUseCase(storageClient, tasksQueueAdapter, userTasksClient, tasksResultsAdapter)
	findNewObjectsUseCase := usecase.NewFindNewObjectsUseCase(linksSearchQueueAdapter, userTasksClient, tasksResultsAdapter)
	// findNewObjectsUseCase := usecase.NewFindNewObjectsUseCase(storageClient, tasksQueueAdapter)

	appLogger.Info("All use cases initialized", nil)

	apiHandlers := rest.NewActualizationHandlers(actualizeActiveObjectsUseCase, actualizeArchivedObjectsUseCase, actualizeObjectByIdUseCase, findNewObjectsUseCase)
	apiServer := rest.NewServer(appConfig.Rest.PORT, apiHandlers, baseLogger)

	// 5. Собираем приложение
	application := &App{
		config:        appConfig,
		apiServer:     apiServer,
		eventProducer: eventProducer,
		logger:        appLogger,    // <-- СОХРАНЯЕМ ЛОГГЕР
		fluentClient:  fluentClient, // <-- СОХРАНЯЕМ КЛИЕНТ ДЛЯ ЗАКРЫТИЯ
	}

	return application, nil
}

// Run запускает все компоненты приложения и управляет их жизненным циклом.
func (a *App) Run() error {
	// Создаем единый контекст для всего приложения для управления graceful shutdown
	appCtx, cancelApp := context.WithCancel(context.Background())
	//defer cancelApp()

	defer func() {
		a.logger.Info("Shutdown sequence initiated...", nil)

		if a.apiServer != nil {
			if err := a.apiServer.Stop(context.Background()); err != nil {
				a.logger.Error("Error during API server shutdown", err, nil)
			}
		}

		if a.eventProducer != nil {
			if err := a.eventProducer.Close(); err != nil {
				a.logger.Error("Error closing event producer", err, nil)
			}
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

	serverErrors := make(chan error, 1)
	go func() {
		a.logger.Info("Starting HTTP server...", port.Fields{"port": a.config.Rest.PORT})
		if err := a.apiServer.Start(); err != nil && err != http.ErrServerClosed {
			serverErrors <- fmt.Errorf("failed to start HTTP server: %w", err)
		}
	}()

	// Ожидание сигнала на завершение или ошибки от одного из компонентов
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	a.logger.Info("Application running. Waiting for signals or server error...", nil)
	select {
	case receivedSignal := <-quit:
		a.logger.Warn("Received OS signal, shutting down...", port.Fields{"signal": receivedSignal.String()})
	case <-appCtx.Done():
		a.logger.Warn("Context was cancelled unexpectedly, shutting down...", nil)
	case err := <-serverErrors:
		a.logger.Error("HTTP server failed to start, shutting down", err, nil)
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
