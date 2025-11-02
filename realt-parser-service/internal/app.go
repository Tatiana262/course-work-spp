package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"realt-parser-service/internal/adapters/realtfetcher"
	postgres_adapter "realt-parser-service/internal/adapters/postgres"
	rabbitmq_adapter "realt-parser-service/internal/adapters/rabbitmq"
	"realt-parser-service/internal/configs"
	"realt-parser-service/internal/constants"
	"realt-parser-service/internal/core/domain"
	"realt-parser-service/internal/core/port"
	usecases_port "realt-parser-service/internal/core/port/usecases"
	"realt-parser-service/internal/core/usecase"
	"realt-parser-service/pkg/postgres"
	"realt-parser-service/pkg/rabbitmq/rabbitmq_common"
	"realt-parser-service/pkg/rabbitmq/rabbitmq_consumer"
	"realt-parser-service/pkg/rabbitmq/rabbitmq_producer"
	"sync"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
)

// App – структура приложения
type App struct {
	config        *configs.AppConfig
	dbPool        *pgxpool.Pool
	eventProducer *rabbitmq_producer.Publisher

	// Use Case, который запускается самим приложением
	fetchRealtLinksUseCase usecases_port.FetchLinksPort

	// Входящие порты (слушатели событий)
	linkEventsListener          port.EventListenerPort
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

	producerCfg := rabbitmq_producer.PublisherConfig{
		Config:                   rabbitmq_common.Config{URL: appConfig.RabbitMQ.URL},
		ExchangeName:             "parser_exchange",
		ExchangeType:             "direct",
		DurableExchange:          true,
		DeclareExchangeIfMissing: true,
	}
	eventProducer, err := rabbitmq_producer.NewPublisher(producerCfg)
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("failed to create event producer: %w", err)
	}
	log.Println("RabbitMQ Event Producer initialized.")

	realtAdapter := realtfetcher.NewRealtFetcherAdapter(
		"https://realt.by/bff/graphql",
	)

	
	log.Println("Unified Realt Fetcher Adapter initialized.")

	linkQueueAdapter, _ := rabbitmq_adapter.NewRabbitMQLinkQueueAdapter(eventProducer, constants.RoutingKeyLinkTasks)
	processedPropertyQueueAdapter, _ := rabbitmq_adapter.NewRabbitMQProcessedPropertyQueueAdapter(eventProducer, constants.RoutingKeyProcessedProperties)
	pgLastRunRepo, _ := postgres_adapter.NewPostgresLastRunRepository(dbPool)

    
	log.Println("All outgoing adapters initialized.")

	// инициализация use-cases
	fetchRealtUseCase := usecase.NewFetchAndEnqueueLinksUseCase(realtAdapter, linkQueueAdapter, pgLastRunRepo, "realt")
	processLinkUseCase := usecase.NewProcessLinkUseCase(realtAdapter, processedPropertyQueueAdapter)
	log.Println("All use cases initialized.")

	// инициализация входящих адаптеров
	linksConsumerCfg := rabbitmq_consumer.ConsumerConfig{
		Config:              rabbitmq_common.Config{URL: appConfig.RabbitMQ.URL},
		QueueName:           constants.QueueLinkTasks,
		RoutingKeyForBind:   constants.RoutingKeyLinkTasks,
		ExchangeNameForBind: "parser_exchange",
		PrefetchCount:       5,
		DurableQueue:        true,
		ConsumerTag:         "link-processor-adapter",
		DeclareQueue:        true,

		EnableRetryMechanism: true,
    
		RetryExchange:        constants.QueueLinkTasks + "_retry_ex",
		RetryQueue:           constants.QueueLinkTasks + "_retry_wait_10s",
		RetryTTL:             10000, 
		
		FinalDLXExchange:     constants.FinalDLXExchange,
		FinalDLQ:             constants.FinalDLQ,
		FinalDLQRoutingKey:   constants.FinalDLQRoutingKey,
		
		MaxRetries:           3,
	}
	linkListener, err := rabbitmq_adapter.NewLinkConsumerAdapter(linksConsumerCfg, processLinkUseCase)
	if err != nil {
		eventProducer.Close()
		dbPool.Close()
		return nil, err
	}
	log.Println("Link Events Listener initialized.")


	// Собираем приложение
	application := &App{
		config:                      appConfig,
		dbPool:                      dbPool,
		eventProducer:               eventProducer,
		fetchRealtLinksUseCase:      fetchRealtUseCase, 
		linkEventsListener:          linkListener,
	}

	return application, nil
}

// StartRealtLinkFetcher запускает процесс сбора ссылок
func (a *App) StartRealtLinkFetcher(ctx context.Context) {
	log.Println("App: Initiating Realt link fetching...")
	searches := constants.GetSearchTasks()

	for _, search := range searches {
		go func(crit domain.SearchCriteria, searchName string) {
			if err := a.fetchRealtLinksUseCase.Execute(ctx, crit); err != nil {
				log.Printf("App: Realt link fetching for '%s' finished with error: %v", searchName, err)
			} else {
				log.Printf("App: Realt link fetching for '%s' finished successfully.", searchName)
			}
		}(search, search.Name)
	}
}

// Run запускает все компоненты приложения и управляет их жизненным циклом
func (a *App) Run() error {
	// Создаем единый контекст для всего приложения для управления graceful shutdown
	appCtx, cancelApp := context.WithCancel(context.Background())
	
	var wg sync.WaitGroup

	defer func() {
		log.Println("App: Shutdown sequence initiated...")

		// Ждем завершения всех запущенных горутин
		log.Println("App: Waiting for background processes to finish...")
		wg.Wait()
		log.Println("App: All background processes finished.")

		if a.linkEventsListener != nil {
			if err := a.linkEventsListener.Close(); err != nil {
				log.Printf("App: Error closing links listener: %v\n", err)
			}
		}
		if a.eventProducer != nil {
			if err := a.eventProducer.Close(); err != nil {
				log.Printf("App: Error closing event producer: %v\n", err)
			}
		}
		if a.dbPool != nil {
			a.dbPool.Close()
			log.Println("App: PostgreSQL pool closed.")
		}
		log.Println("Application shut down gracefully.")
	}()

	log.Println("Application is starting...")

	// Запускаем сборщик ссылок, передавая ему главный контекст
	a.StartRealtLinkFetcher(appCtx)

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
	go startListener("Links Events Listener", a.linkEventsListener)

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