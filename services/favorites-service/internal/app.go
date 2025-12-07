package internal

import (
	"context"
	logger_adapter "favorites-service/internal/adapters/logger"
	postgres_adapter "favorites-service/internal/adapters/postgres"
	"favorites-service/internal/adapters/rest"
	storage_api_client "favorites-service/internal/adapters/storage_client"
	"favorites-service/internal/configs"
	"favorites-service/internal/core/port"
	"favorites-service/internal/core/usecase"
	"fmt"
	"log"
	fluentlogger "real-estate-system/pkg/fluent_logger"
	"real-estate-system/pkg/postgres"
	"strings"

	// "log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	config    *configs.AppConfig
	dbPool    *pgxpool.Pool
	apiServer *rest.Server

	fluentClient *fluent.Fluent
	logger       port.LoggerPort
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

	// 1. Инициализация низкоуровневых зависимостей
	dbPool, err := postgres.NewClient(context.Background(), postgres.Config{DatabaseURL: appConfig.Database.URL})
	if err != nil {
		appLogger.Error("Failed to connect to PostgreSQL", err, nil)
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	appLogger.Info("Successfully connected to PostgreSQL pool!", nil)

	postgresStorageAdapter, err := postgres_adapter.NewPostgresFavoritesRepository(dbPool)
	if err != nil {
		appLogger.Error("Failed to create postgres favorites repository", err, nil)
		dbPool.Close()
		return nil, fmt.Errorf("failed to create postgres storage adapter: %w", err)
	}

	storageClient := storage_api_client.NewStorageServiceAPIClient(appConfig.ApiClient.STORAGE_PORT)
	appLogger.Info("All persistence and service adapters initialized.", nil)

	// ИНИЦИАЛИЗАЦИЯ USE CASES (ядра бизнес-логики)
	addToFavoritesUseCase := usecase.NewAddToFavoritesUseCase(postgresStorageAdapter)
	removeFromFavoritesUseCase := usecase.NewRemoveFromFavoritesUseCase(postgresStorageAdapter)
	getUserFavoritesUseCase := usecase.NewGetUserFavoritesUseCase(postgresStorageAdapter, storageClient)
	appLogger.Info("REST API server configured.", nil)

	// REST API Server
	apiHandlers := rest.NewFavoritesHandler(addToFavoritesUseCase, removeFromFavoritesUseCase, getUserFavoritesUseCase)
	apiServer := rest.NewServer(appConfig.Rest.PORT, apiHandlers, baseLogger)

	// 5. Собираем приложение
	application := &App{
		config:    appConfig,
		dbPool:    dbPool,
		apiServer: apiServer,

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

	defer func() {
		a.logger.Info("Shutdown sequence initiated...", nil)

		if a.apiServer != nil {
			if err := a.apiServer.Stop(context.Background()); err != nil {
				a.logger.Error("Error during API server shutdown", err, nil)
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

	serverErrors := make(chan error, 1)
	go func() {
		a.logger.Info("Starting HTTP server...", port.Fields{"port": a.config.Rest.PORT})
		if err := a.apiServer.Start(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
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
		a.logger.Error("Server failed to start, shutting down", err, nil)
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