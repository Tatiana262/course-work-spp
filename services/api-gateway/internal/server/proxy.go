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


func CreateSSEProxy(targetURL, pathPrefix string) http.Handler {
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("Invalid target URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// ДЛЯ SSE
	// -1 означает: сбрасывать данные клиенту МГНОВЕННО после получения от бэкенда.
	// Не ждать накопления буфера.
	proxy.FlushInterval = -1 

	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host
		req.URL.Path = pathPrefix + req.URL.Path

		traceID := contextkeys.TraceIDFromContext(req.Context())
		if traceID != "" {
			req.Header.Set("X-Trace-ID", traceID)
		}
		
		// Для SSE важно, чтобы Connection не закрывался
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
	}

	return proxy
}