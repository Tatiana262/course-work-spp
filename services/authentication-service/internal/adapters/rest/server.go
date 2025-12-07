package rest

import (
	core_port "authentication-service/internal/core/port"
	"context"
	"fmt"
	// "log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server - наш REST API сервер.
type Server struct {
	httpServer *http.Server
	logger     core_port.LoggerPort
}

// NewServer создает новый экземпляр сервера.
func NewServer(port string, handlers *AuthHandlers, baseLogger core_port.LoggerPort) *Server {
	r := chi.NewRouter()

	// serverLogger := baseLogger.WithFields(core_port.Fields{"component": "rest_server"})

	// Middleware
	r.Use(LoggerMiddleware(baseLogger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	// Роуты
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", handlers.Register)
		r.Post("/login", handlers.Login)
		r.Post("/validate", handlers.ValidateToken) // Эндпоинт для проверки токена
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
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