package server

import (
	"api-gateway/internal/auth"
	"api-gateway/internal/configs"
	"api-gateway/internal/port"
	// "log"
	"net/http"
	// "time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// NewServer создает и настраивает главный роутер и HTTP-сервер.
func NewServer(cfg *configs.Config, authClient *auth.Client, baseLogger port.LoggerPort) *http.Server {
	r := chi.NewRouter()

	// Стандартные middleware
	r.Use(middleware.RealIP, LoggerMiddleware(baseLogger), middleware.Recoverer)
	// r.Use(middleware.Timeout(60 * time.Second))
	
	r.Use(cors.Handler(cors.Options{
        // AllowedOrigins - список доменов, с которых разрешены запросы
        AllowedOrigins:   []string{"http://localhost:5173"},
        
        // AllowedMethods - список разрешенных HTTP-методов.
        AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        
        // AllowedHeaders - список разрешенных заголовков в запросе
        AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
                
        // AllowCredentials - разрешает отправку cookies и Authorization хедера
        AllowCredentials: true,
        
        // MaxAge - на сколько секунд браузер может кэшировать результат preflight-запроса
        MaxAge:           300, // 5 минут
    }))

	// Создаем middleware для аутентификации
	authMiddleware := NewAuthMiddleware(authClient)

	// --- Префикс для всех внутренних API ---
	const internalApiPrefix = "/api/v1"

	// --- Публичные маршруты ---
	r.Group(func(r chi.Router) {
		// /auth/* -> authentication-service/api/v1/auth/*
		r.Mount("/auth", CreateProxy(cfg.AuthServiceURL, internalApiPrefix))

		// /objects/* -> storage-service/api/v1/objects/*
		r.Mount("/objects", CreateProxy(cfg.StorageServiceURL, internalApiPrefix))
		r.Mount("/filters/options", CreateProxy(cfg.StorageServiceURL, internalApiPrefix))
		r.Mount("/dictionaries", CreateProxy(cfg.StorageServiceURL, internalApiPrefix))
	})

	// --- Приватные маршруты (для всех авторизованных) ---
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		
		// /favorites/* -> favorites-service/api/v1/favorites/*
		r.Mount("/favorites", CreateProxy(cfg.FavoritesServiceURL, internalApiPrefix))
		
		// /actualize/object/* -> actualization-service/api/v1/actualize/object/*
		//Сначала более специфичный
		r.Mount("/actualize/object", CreateProxy(cfg.ActualizationServiceURL, internalApiPrefix))
		r.Mount("/tasks/subscribe", CreateSSEProxy(cfg.TasksServiceURL, internalApiPrefix))
	})

	// --- Приватные маршруты (только для админов) ---
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.Use(authMiddleware.RequireRole("admin"))

		// log.Println("middlewares passed")

		// /actualize/* -> actualization-service/api/v1/actualize/*
		// после более специфичного /actualize/object
		r.Mount("/actualize", CreateProxy(cfg.ActualizationServiceURL, internalApiPrefix))
		r.Mount("/tasks", CreateProxy(cfg.TasksServiceURL, internalApiPrefix))
	})


	return &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}
}