package rest

import (
	core_ports "actualization-service/internal/core/port"
	"context"
	"fmt"
	"net/http"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)


type Server struct {
	httpServer *http.Server
	logger     core_ports.LoggerPort
}

func NewServer(port string, handlers *ActualizationHandlers, baseLogger core_ports.LoggerPort) *Server {
	r := chi.NewRouter()

	r.Use(LoggerMiddleware(baseLogger)) // Логирует каждый запрос (метод, путь, время выполнения)
	r.Use(middleware.Recoverer)         // Перехватывает паники и возвращает 500 ошибку, чтобы сервер не упал

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/actualize", func(r chi.Router) {

			r.Use(AuthMiddleware)

			r.Post("/active", handlers.HandleActualizeActiveObjects)
			r.Post("/archived", handlers.HandleActualizeArchivedObjects)
			r.Post("/object", handlers.HandleActualizeObjectByID)
			r.Post("/new-objects", handlers.HandleFindNewObjects)
		})

	})

	return &Server{
		httpServer: &http.Server{
			Addr:    ":" + port,
			Handler: r,
		},
		logger: baseLogger,
	}
}

// Start запускает HTTP-сервер
func (s *Server) Start() error {

	s.logger.Info("Starting REST API server", core_ports.Fields{"address": s.httpServer.Addr})
	// ListenAndServe будет работать, пока не получит ошибку или команду Shutdown
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
