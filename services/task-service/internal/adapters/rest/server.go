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

type Server struct {
	httpServer *http.Server
	logger     core_port.LoggerPort
}


func NewServer(port string, handlers *TaskHandler, baseLogger core_port.LoggerPort) *Server {
	r := chi.NewRouter()


	// Общие middleware
	r.Use(LoggerMiddleware(baseLogger))
	r.Use(middleware.Recoverer)

	r.Route("/api/v1/tasks", func(r chi.Router) {
		
		// для других сервисов (без проверки userID)
		r.Post("/", handlers.CreateTask)
		r.Put("/{taskID}", handlers.UpdateTask)

		// эндпоинты для пользователей (через API Gateway)
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware)

			// GET /api/v1/tasks - получить список своих задач
			r.Get("/", handlers.GetTasksList)
			
			// GET /api/v1/tasks/subscribe - подписаться на обновления своих задач
			r.Get("/subscribe", handlers.SubscribeToTasks)
			
			// GET /api/v1/tasks/{taskID} - получить детали задачи
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

// Start запускает HTTP-сервер
func (s *Server) Start() error {
	s.logger.Info("Starting REST API server", core_port.Fields{"address": s.httpServer.Addr})
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("Could not start server", err, nil)
		return fmt.Errorf("could not start server: %w", err)
	}
	return nil
}

// Stop корректно останавливает сервер
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping REST API server...", nil)
	return s.httpServer.Shutdown(ctx)
}