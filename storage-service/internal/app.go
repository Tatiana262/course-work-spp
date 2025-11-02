package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	postgres_adapter "storage-service/internal/adapters/postgres"	
	"storage-service/internal/adapters/rest"
	"storage-service/internal/configs"
	
	"storage-service/internal/core/port"
	"storage-service/internal/core/usecase"
	"storage-service/pkg/postgres"
	"storage-service/internal/constants"
	rabbitmq_adapter "storage-service/internal/adapters/rabbitmq"
	"storage-service/pkg/rabbitmq/rabbitmq_common"
	"storage-service/pkg/rabbitmq/rabbitmq_consumer"
	"sync"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
)

// App – структура приложения
type App struct {
	config        *configs.AppConfig
	dbPool        *pgxpool.Pool
	apiServer	  *rest.Server

	processedPropEventsListener port.EventListenerPort
}

// NewApp создает новый экземпляр приложения
func NewApp() (*App, error) {
	appConfig, err := configs.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading application configuration: %w", err)
	}

	dbPool, err := postgres.NewClient(context.Background(), postgres.Config{DatabaseURL: appConfig.Database.URL})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	log.Println("Successfully connected to PostgreSQL pool!")
	

	postgresStorageAdapter, err := postgres_adapter.NewPostgresStorageAdapter(dbPool)
    if err != nil {
		dbPool.Close()       
        return nil, fmt.Errorf("failed to create postgres storage adapter: %w", err)
    }
    
	log.Println("All outgoing adapters initialized.")

	// инициализация use-cases
	savePropertyUseCase := usecase.NewSavePropertyUseCase(postgresStorageAdapter)
	getActiveObjectsUseCase := usecase.NewGetActiveObjectsUseCase(postgresStorageAdapter)
	getArchivedObjectsUseCase := usecase.NewGetArchivedObjectsUseCase(postgresStorageAdapter)
	getObjectByIDUseCase := usecase.NewGetObjectByIDUseCase(postgresStorageAdapter)
	log.Println("All use cases initialized.")

	// инициализация входящих адаптеров
	processedConsumerCfg := rabbitmq_consumer.ConsumerConfig{
		Config:              rabbitmq_common.Config{URL: appConfig.RabbitMQ.URL},
		QueueName:           constants.QueueProcessedProperties,
		DurableQueue:        true,
		ExchangeNameForBind: "parser_exchange",
		RoutingKeyForBind:   constants.RoutingKeyProcessedProperties,
		PrefetchCount:       1,
		ConsumerTag:         "property-saver-adapter",
		DeclareQueue:        true,

		EnableRetryMechanism: true,

		RetryExchange:        constants.QueueProcessedProperties + "_retry_ex",
		RetryQueue:           constants.QueueProcessedProperties + "_retry_wait_10s",
		RetryTTL:             10000, 
		 
		FinalDLXExchange:     constants.FinalDLXExchange,
		FinalDLQ:             constants.FinalDLQ,
		FinalDLQRoutingKey:   constants.FinalDLQRoutingKey,
		
		MaxRetries:           3,
	}
	processedPropListener, err := rabbitmq_adapter.NewProcessedPropertyConsumerAdapter(processedConsumerCfg, savePropertyUseCase)
	if err != nil {
		dbPool.Close()
		return nil, err
	}
	log.Println("Processed Property Events Listener initialized.")

	// REST API Server
    apiHandlers := rest.NewPropertyHandlers(getActiveObjectsUseCase, getArchivedObjectsUseCase, getObjectByIDUseCase)
    apiServer := rest.NewServer(appConfig.Rest.PORT, apiHandlers)

	// Собираем приложение
	application := &App{
		config:                      appConfig,
		dbPool:                      dbPool,	
		apiServer:					 apiServer,	
		processedPropEventsListener: processedPropListener,
	}

	return application, nil
}

// Run запускает все компоненты приложения и управляет их жизненным циклом
func (a *App) Run() error {
	// единый контекст для всего приложения для управления graceful shutdown
	appCtx, cancelApp := context.WithCancel(context.Background())

	// для ожидания завершения всех фоновых задач
	var wg sync.WaitGroup

	defer func() {
		log.Println("App: Shutdown sequence initiated...")

		// Ждем завершения всех запущенных (слушателей)
		log.Println("App: Waiting for background processes to finish...")
		wg.Wait()
		log.Println("App: All background processes finished.")

		// закрываем ресурсы
		if a.processedPropEventsListener != nil {
			if err := a.processedPropEventsListener.Close(); err != nil {
				log.Printf("App: Error closing processed properties listener: %v\n", err)
			}
		}
		
		if a.apiServer != nil {
			if err := a.apiServer.Stop(context.Background()); err != nil {
				log.Printf("App: Error closing api server: %v\n", err)
			}
		}
		
		if a.dbPool != nil {
			a.dbPool.Close()
			log.Println("App: PostgreSQL pool closed.")
		}
		log.Println("Application shut down gracefully.")
	}()

	log.Println("Application is starting...")

	consumerErrors := make(chan error, 1)

	// Функция для запуска слушателей
	startListener := func(name string, listener port.EventListenerPort) {
		defer wg.Done()
		log.Printf("App: Starting %s...", name)
		if err := listener.Start(appCtx); err != nil {
			log.Printf("App: %s stopped with an unexpected error: %v", name, err)
			consumerErrors <- fmt.Errorf("%s error: %w", name, err)
		} else {
			log.Printf("App: %s stopped gracefully due to context cancellation.", name)
		}
	}

	wg.Add(1)
	go startListener("Processed Property Events Listener", a.processedPropEventsListener)

	go func() {
        log.Printf("Starting HTTP server on port %s...", a.config.Rest.PORT)
        if err := a.apiServer.Start(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Failed to start HTTP server: %v", err)
        }
    }()

	// Ожидание сигнала на завершение или ошибки от одного из компонентов
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Application running. Waiting for signals or consumer error...")
	select {
	case receivedSignal := <-quit:
		log.Printf("App: Received signal: %s. Shutting down...\n", receivedSignal)
	case err := <-consumerErrors:
		log.Printf("App: A critical component failed: %v. Shutting down...\n", err)
	case <-appCtx.Done():
		log.Println("App: Context was cancelled unexpectedly. Shutting down...")
	}

	// Инициируем graceful shutdown, отменяя главный контекст
	cancelApp()

	return nil
}