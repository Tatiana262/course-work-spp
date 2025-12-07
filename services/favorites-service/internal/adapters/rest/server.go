package rest

import (
	"context"
	core_port "favorites-service/internal/core/port"
	"fmt"
	// "log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Handlers - интерфейс, описывающий все наши обработчики.
// type Handlers interface {
// 	GetUserFavorites(w http.ResponseWriter, r *http.Request)
// 	AddToFavorites(w http.ResponseWriter, r *http.Request)
// 	RemoveFromFavorites(w http.ResponseWriter, r *http.Request)
// }

// Server - наш REST API сервер.
type Server struct {
	httpServer *http.Server
	logger     core_port.LoggerPort
}

// NewServer создает новый экземпляр сервера.
func NewServer(port string, handlers *FavoritesHandler, baseLogger core_port.LoggerPort) *Server {
	r := chi.NewRouter()

	// serverLogger := baseLogger.WithFields(core_port.Fields{"component": "rest_server"})

	// Middleware
	r.Use(LoggerMiddleware(baseLogger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	// Группа роутов для нашего API. Все они будут приватными.
	r.Route("/api/v1/favorites", func(r chi.Router) {
		// Middleware для аутентификации ( API Gateway)
		r.Use(AuthMiddleware)

		r.Get("/", handlers.GetUserFavorites)
		r.Post("/", handlers.AddToFavorites)
		r.Delete("/{masterObjectID}", handlers.RemoveFromFavorites)
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