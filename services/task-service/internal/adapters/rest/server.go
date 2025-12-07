package rest

import (
	"context"
	"fmt"
	// "log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	core_port "task-service/internal/core/port"
)

// Server - наш REST API сервер для task-service.
type Server struct {
	httpServer *http.Server
	logger     core_port.LoggerPort
}


func NewServer(port string, handlers *TaskHandler, baseLogger core_port.LoggerPort) *Server {
	r := chi.NewRouter()

	// serverLogger := baseLogger.WithFields(core_port.Fields{"component": "rest_server"})

	// Общие middleware
	r.Use(LoggerMiddleware(baseLogger))
	r.Use(middleware.Recoverer)

	// Роутинг для API v1
	r.Route("/api/v1/tasks", func(r chi.Router) {
		
		// --- Публичные/Внутренние роуты (без проверки userID) ---
		// Эти эндпоинты вызываются другими сервисами (например, actualization-service),
		// которые могут действовать не от имени конкретного пользователя.
		// Мы доверяем нашей внутренней сети.
		r.Post("/", handlers.CreateTask)
		r.Put("/{taskID}", handlers.UpdateTask)


		// --- Приватные роуты (требуют userID из заголовка) ---
		// Эти эндпоинты вызываются от имени пользователя (через API Gateway).
		r.Group(func(r chi.Router) {
			// Применяем нашу AuthMiddleware ко всей группе.
			r.Use(AuthMiddleware)

			// GET /api/v1/tasks - получить список своих задач
			r.Get("/", handlers.GetTasksList)
			
			// GET /api/v1/tasks/subscribe - подписаться на обновления своих задач
			r.Get("/subscribe", handlers.SubscribeToTasks)
			
			// GET /api/v1/tasks/{taskID} - получить детали своей задачи
			// (проверка, что пользователь может смотреть именно эту задачу,
			// должна быть внутри хендлера или use case).
			r.Get("/{taskID}", handlers.GetTaskByID)
		})
	})

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
	}

	return &Server{
		httpServer: srv,
		logger:     baseLogger,
	}
}

// Start запускает HTTP-сервер.
func (s *Server) Start() error {
	s.logger.Info("Starting REST API server", core_port.Fields{"address": s.httpServer.Addr})
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("Could not start server", err, nil)
		return fmt.Errorf("could not start server: %w", err)
	}
	return nil
}

// Stop корректно останавливает сервер.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping REST API server...", nil)
	return s.httpServer.Shutdown(ctx)
}