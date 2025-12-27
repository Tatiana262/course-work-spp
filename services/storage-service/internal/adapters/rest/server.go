package rest

import (
	"context"
	"net/http"
	core_port "storage-service/internal/core/port"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

type Server struct {
    httpServer *http.Server	
    logger     core_port.LoggerPort
}

func NewServer(port string, 
    actualiztion_handlers *ActualiztionObjectsHandler, 
    get_info_handlers *GetInfoHandler,
    filters_handlers *FilterHandler,
    baseLogger core_port.LoggerPort) *Server {

    r := chi.NewRouter()


    r.Use(LoggerMiddleware(baseLogger), middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/active-objects", actualiztion_handlers.GetActiveObjects)
        r.Get("/archived-objects", actualiztion_handlers.GetArchivedObjects)
        r.Get("/object", actualiztion_handlers.GetObjectsByMasterID)

        r.Post("/objects/best-by-master-ids", get_info_handlers.GetBestByMasterIDs)

        // роуты для пользователей
        r.Get("/objects", get_info_handlers.FindObjects)
        r.Get("/objects/{objectID}", get_info_handlers.GetObjectDetails)

        r.Get("/filters/options", filters_handlers.GetFilterOptions)
        r.Get("/dictionaries", filters_handlers.GetDictionaries)
        r.Get("/stats", actualiztion_handlers.GetActualizationStats)
	})
    
    
    return &Server{
        httpServer: &http.Server{
            Addr:    ":" + port,
            Handler: r,
        },
        logger: baseLogger,
    }
}

func (s *Server) Start() error {
    s.logger.Info("Starting REST server", core_port.Fields{"address": s.httpServer.Addr})
    return s.httpServer.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
    s.logger.Info("Stopping REST server...", nil)
    return s.httpServer.Shutdown(ctx)
}