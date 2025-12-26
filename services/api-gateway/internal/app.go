package internal

import (
	"context"
	"fmt"
	"log"
	"strings"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	logger_adapter "api-gateway/internal/adapter/logger"
	"api-gateway/internal/auth"
	"api-gateway/internal/configs"
	"api-gateway/internal/port"
	"api-gateway/internal/server"
	fluentlogger "real-estate-system/pkg/fluent_logger"
	"github.com/fluent/fluent-logger-golang/fluent"
)

// App - основная структура приложения
type App struct {
	httpServer   *http.Server
	logger       port.LoggerPort
	fluentClient *fluent.Fluent
}

// NewApp создает и настраивает все компоненты приложения
func NewApp() (*App, error) {

	appConfig, err := configs.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading application configuration: %w", err)
	}

	// инициализация логеров
	var activeLoggers []port.LoggerPort

	slogCfg := logger_adapter.SlogConfig{
		Level:    parseLogLevel(appConfig.StdoutLogger.Level),
		IsJSON:   false, // текстовый формат
		UseColor: true,
	}
	stdoutLogger := logger_adapter.NewSlogAdapter(slogCfg)
	activeLoggers = append(activeLoggers, stdoutLogger)

	// Добавляем Fluent Bit логгер, если он включен в конфигурации
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

	// базовый логер приложения с контекстом
	baseLogger := multiLogger.WithFields(port.Fields{
		"service_name": appConfig.AppName,
		// "service_version": "1.0.0",
	})

	appLogger := baseLogger.WithFields(port.Fields{"component": "app"})
	appLogger.Debug("Logger system initialized", port.Fields{
		"active_loggers": len(activeLoggers), "fluent_enabled": appConfig.FluentBit.Enabled,
	})

	// Инициализация исходящих адаптеров (клиентов)
	authClient := auth.NewClient(appConfig.AuthServiceURL)
	appLogger.Debug("Auth client initialized", port.Fields{"target_url": appConfig.AuthServiceURL})

	// Инициализация входящего адаптера (веб-сервера)
	// Передаем ему конфигурацию и созданного клиента
	httpServer := server.NewServer(appConfig, authClient, baseLogger)

	return &App{
		httpServer:   httpServer,
		logger:       appLogger,
		fluentClient: fluentClient,
	}, nil
}

// Run запускает приложение и управляет его жизненным циклом
func (a *App) Run() error {
	// Запускаем HTTP-сервер в отдельной горутине
	go func() {
		a.logger.Info("API Gateway is listening", port.Fields{"port": a.httpServer.Addr})
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("Failed to start API Gateway", err, nil)
			os.Exit(1) // Если сервер не может запуститься, это фатально
		}
	}()

	// Настройка Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	a.logger.Debug("API Gateway is shutting down...", port.Fields{"signal": sig.String()})

	// Создаем контекст с таймаутом для завершения
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Корректно останавливаем сервер
	if err := a.httpServer.Shutdown(ctx); err != nil {
		a.logger.Error("API Gateway shutdown failed", err, nil)
		os.Exit(1)
	}

	a.logger.Info("API Gateway shut down gracefully.", nil)

	a.logger.Info("Application shut down gracefully.", nil)
	if a.fluentClient != nil {
		if err := a.fluentClient.Close(); err != nil {
			fmt.Printf("ERROR: Error closing fluent client: %v\n", err)
		}
	}

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
