package rest

import (
    "context"
    "net/http"

	"github.com/go-chi/chi/v5"
)

type Server struct {
    httpServer *http.Server	
}

func NewServer(port string, handlers *ActualiztionObjectsHandler) *Server {
    r := chi.NewRouter()

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/active-objects", handlers.GetActiveObjects)
        r.Get("/archived-objects", handlers.GetArchivedObjects)
        r.Get("/objects", handlers.GetObjectByID)
	})
    
    
    return &Server{
        httpServer: &http.Server{
            Addr:    ":" + port,
            Handler: r,
        },
    }
}

func (s *Server) Start() error {
    return s.httpServer.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
    return s.httpServer.Shutdown(ctx)
}