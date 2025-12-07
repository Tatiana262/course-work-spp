package internal

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	fluentlogger "real-estate-system/pkg/fluent_logger"
	"real-estate-system/pkg/postgres"
	"real-estate-system/pkg/rabbitmq/rabbitmq_common"
	"real-estate-system/pkg/rabbitmq/rabbitmq_consumer"
	"strings"
	"sync"
	"syscall"
	logger_adapter "task-service/internal/adapters/logger"
	"task-service/internal/adapters/notifier"
	postgres_adapter "task-service/internal/adapters/postgres"
	rabbitmq_adapter "task-service/internal/adapters/rabbitmq"
	"task-service/internal/adapters/rest"
	"task-service/internal/configs"
	"task-service/internal/constants"
	"task-service/internal/core/port"
	"task-service/internal/core/usecase"

	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	config                         *configs.AppConfig
	dbPool                         *pgxpool.Pool
	apiServer                      *rest.Server
	resultsListener                port.EventListenerPort
	tasksCompletionResultsListener port.EventListenerPort

	logger       port.LoggerPort
	fluentClient *fluent.Fluent
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

	taskRepo, err := postgres_adapter.NewPostgresTaskRepository(dbPool)
	if err != nil {
		appLogger.Error("Failed to create postgres task repository", err, nil)
		dbPool.Close()
		return nil, fmt.Errorf("failed to create postgres storage adapter: %w", err)
	}

	sseNotifier := notifier.NewSSENotifier(baseLogger)
	appLogger.Info("SSE Notifier initialized.", nil)

	// ИНИЦИАЛИЗАЦИЯ USE CASES (ядра бизнес-логики)
	createTaskUC := usecase.NewCreateTaskUseCase(taskRepo, sseNotifier)
	updateTaskUC := usecase.NewUpdateTaskStatusUseCase(taskRepo, sseNotifier)
	getTaskByIdUC := usecase.NewGetTaskByIdUseCase(taskRepo)
	getTasksUC := usecase.NewGetTasksListUseCase(taskRepo)
	processResultUC := usecase.NewProcessTaskResultUseCase(taskRepo, sseNotifier)
	completeTaskUC := usecase.NewCompleteTaskUseCase(taskRepo, sseNotifier)
	appLogger.Info("All use cases initialized.", nil)

	// REST API Server
	apiHandlers := rest.NewTaskHandler(createTaskUC, updateTaskUC, getTaskByIdUC, getTasksUC, processResultUC, sseNotifier)
	apiServer := rest.NewServer(appConfig.Rest.PORT, apiHandlers, baseLogger)
	appLogger.Info("REST API server configured.", nil)

	// RabbitMQ Consumer для результатов
	consumerCfg := rabbitmq_consumer.ConsumerConfig{
		Config:              rabbitmq_common.Config{URL: appConfig.RabbitMQ.URL},
		QueueName:           constants.QueueTaskResults,
		RoutingKeyForBind:   constants.RoutingKeyTaskResults,
		ExchangeNameForBind: "parser_exchange",
		PrefetchCount:       5,
		DurableQueue:        true,
		ConsumerTag:         "task-results-processor-adapter",
		DeclareQueue:        true,

		// --- НОВЫЕ НАСТРОЙКИ ---
		// 1. Включаем сам механизм
		EnableRetryMechanism: true,

		// 2. Настраиваем "сателлиты" для этой конкретной очереди.
		// Используем имя основной очереди как префикс для уникальности.
		RetryExchange: constants.QueueTaskResults + "_retry_ex",
		RetryQueue:    constants.QueueTaskResults + "_retry_wait_10s",
		RetryTTL:      10000, // 10 секунд в миллисекундах

		// 3. Указываем общую "свалку" для сообщений, исчерпавших все попытки.
		FinalDLXExchange:   constants.FinalDLXExchange,
		FinalDLQ:           constants.FinalDLQ,
		FinalDLQRoutingKey: constants.FinalDLQRoutingKey,

		// 4. Задаем количество ретраев (помимо первой попытки).
		MaxRetries: 3,
	}

	resultsListener, err := rabbitmq_adapter.NewResultsConsumerAdapter(consumerCfg, processResultUC, baseLogger, connManager)
	if err != nil {
		appLogger.Error("Failed to create results consumer", err, nil)
		dbPool.Close()
		return nil, fmt.Errorf("failed to create results consumer adapter: %w", err)
	}

	consumerTaskCompletionResultsCfg := rabbitmq_consumer.ConsumerConfig{
		Config:              rabbitmq_common.Config{URL: appConfig.RabbitMQ.URL},
		QueueName:           constants.QueueTaskCompletionResults,
		RoutingKeyForBind:   constants.RoutingKeyTaskCompletionResults,
		ExchangeNameForBind: "parser_exchange",
		PrefetchCount:       5,
		DurableQueue:        true,
		ConsumerTag:         "task-completion-results-processor-adapter",
		DeclareQueue:        true,

		// --- НОВЫЕ НАСТРОЙКИ ---
		// 1. Включаем сам механизм
		EnableRetryMechanism: true,

		// 2. Настраиваем "сателлиты" для этой конкретной очереди.
		// Используем имя основной очереди как префикс для уникальности.
		RetryExchange: constants.QueueTaskCompletionResults + "_retry_ex",
		RetryQueue:    constants.QueueTaskCompletionResults + "_retry_wait_10s",
		RetryTTL:      10000, // 10 секунд в миллисекундах

		// 3. Указываем общую "свалку" для сообщений, исчерпавших все попытки.
		FinalDLXExchange:   constants.FinalDLXExchange,
		FinalDLQ:           constants.FinalDLQ,
		FinalDLQRoutingKey: constants.FinalDLQRoutingKey,

		// 4. Задаем количество ретраев (помимо первой попытки).
		MaxRetries: 3,
	}

	tasksCompletionResultsListener, err := rabbitmq_adapter.NewTaskCompletionResultsConsumerAdapter(consumerTaskCompletionResultsCfg, completeTaskUC, baseLogger, connManager)
	if err != nil {
		appLogger.Error("Failed to create task completion consumer", err, nil)
		resultsListener.Close()
		dbPool.Close()
		return nil, fmt.Errorf("failed to create results consumer adapter: %w", err)
	}
	appLogger.Info("All RabbitMQ listeners initialized.", nil)

	// 5. Собираем приложение
	application := &App{
		config:                         appConfig,
		dbPool:                         dbPool,
		apiServer:                      apiServer,
		resultsListener:                resultsListener,
		tasksCompletionResultsListener: tasksCompletionResultsListener,
		logger:                         appLogger,
		fluentClient:                   fluentClient,
	}

	return application, nil

}

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
		if a.resultsListener != nil {
			if err := a.resultsListener.Close(); err != nil {
				a.logger.Error("Error closing results listener", err, nil)
			}
		}

		if a.tasksCompletionResultsListener != nil {
			if err := a.tasksCompletionResultsListener.Close(); err != nil {
				a.logger.Error("Error closing task completion listener", err, nil)
			}
		}

		if a.dbPool != nil {
			a.dbPool.Close()
			a.logger.Info("PostgreSQL pool closed.", nil)
		}

		a.logger.Info("Application shut down gracefully.", nil)
		if a.fluentClient != nil {
			if err := a.fluentClient.Close(); err != nil {
				fmt.Printf("ERROR: Error closing fluent client: %v\n", err)
			}
		}
	}()

	a.logger.Info("Application is starting...", nil)

	errorsCh := make(chan error, 1)

	go func() {
		a.logger.Info("Starting HTTP server...", port.Fields{"port": a.config.Rest.PORT})
		if err := a.apiServer.Start(); err != nil && err != http.ErrServerClosed {
			errorsCh <- fmt.Errorf("HTTP server start error: %w", err)
		}
	}()

	// Функция-хелпер для запуска слушателей
	startListener := func(name string, listener port.EventListenerPort) {
		defer wg.Done()
		listenerLogger := a.logger.WithFields(port.Fields{"listener": name})
		listenerLogger.Info("Starting listener...", nil)

		if err := listener.Start(appCtx); err != nil {
			listenerLogger.Error("Listener stopped with an unexpected error", err, nil)
			errorsCh <- fmt.Errorf("%s error: %w", name, err)
		} else {
			listenerLogger.Info("Listener stopped gracefully.", nil)
		}
	}

	wg.Add(2)
	go startListener("Results Events Listener", a.resultsListener)
	go startListener("Task Completion Results Events Listener", a.tasksCompletionResultsListener)

	// Ожидание сигнала на завершение или ошибки от одного из компонентов
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	a.logger.Info("Application running. Waiting for signals or component error...", nil)
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