package server

import (
	"api-gateway/internal/contextkeys"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	// "strings"
)

// CreateProxy создает универсальный обратный прокси.
// Он берет путь изначального запроса и добавляет к нему префикс.
func CreateProxy(targetURL, pathPrefix string) http.Handler {
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("Invalid target URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(req *http.Request) {
		// Стандартная настройка
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host

		// ---> ВОТ УНИФИЦИРОВАННАЯ ЛОГИКА <---
		// Берем оригинальный путь (например, /objects/123-abc?query=1)
		// и добавляем к нему префикс (например, /api/v1)
		// ВАЖНО: req.URL.Path не содержит query-параметров, они в req.URL.RawQuery
		req.URL.Path = pathPrefix + req.URL.Path

		// 1. Извлекаем trace_id из контекста входящего запроса.
		traceID := contextkeys.TraceIDFromContext(req.Context())

		// 2. Устанавливаем его как заголовок в исходящем запросе.
		if traceID != "" {
			req.Header.Set("X-Trace-ID", traceID)
		}
	}

	return proxy
}


// func createStorageProxy(targetURL string) http.Handler {
// 	target, _ := url.Parse(targetURL)
// 	proxy := httputil.NewSingleHostReverseProxy(target)

// 	proxy.Director = func(req *http.Request) {
// 		req.URL.Scheme = target.Scheme
// 		req.URL.Host = target.Host
// 		req.Host = target.Host
		
//         // ---- Вся магия здесь ----
//         // Исходный путь: /api/objects/some/path?query=1
//         // Мы хотим превратить его в: /api/v1/objects/some/path?query=1
        
//         // 1. Убираем префикс, который "съел" Mount
//         // chi.RouteContext(req.Context()).RoutePath вернет "/api/objects/*"
//         // Нам нужно отрезать статическую часть "/api/objects"
//         trimmedPath := strings.TrimPrefix(req.URL.Path, "/api/objects")
        
//         // 2. Добавляем новый префикс целевого сервиса
//         req.URL.Path = "/api/v1/objects" + trimmedPath
// 	}

// 	return proxy
// }