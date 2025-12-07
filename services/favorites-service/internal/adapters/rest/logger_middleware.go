package rest

import (
	"favorites-service/internal/contextkeys"
	"favorites-service/internal/core/port"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// LoggerMiddleware создает контекстный логгер для каждого запроса.
func LoggerMiddleware(logger port.LoggerPort) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем trace_id от API Gateway. Если его нет, генерируем (fallback).
			traceID := r.Header.Get("X-Trace-ID") // Используем X-Trace-ID для единообразия
			if _, err := uuid.Parse(traceID); err != nil {
				traceID = uuid.New().String()
			}

			// "Чистый" логгер для передачи в use case
			coreLogger := logger.WithFields(port.Fields{"trace_id": traceID})

			// HTTP-логгер для логов самого middleware
			httpLogger := coreLogger.WithFields(port.Fields{
				"http_method": r.Method,
				"http_path":   r.URL.Path,
				"remote_addr": r.RemoteAddr,
			})
			
			// Кладем в контекст и логгер, и trace_id
			ctx := r.Context()
			ctx = contextkeys.ContextWithLogger(ctx, coreLogger)
			ctx = contextkeys.ContextWithTraceID(ctx, traceID)
			
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			startTime := time.Now()
			
			httpLogger.Info("Request started", nil)
			
			next.ServeHTTP(ww, r.WithContext(ctx))
			
			httpLogger.Info("Request finished", port.Fields{
				"status_code": ww.Status(),
				"bytes_written": ww.BytesWritten(),
				"duration_ms":   time.Since(startTime).Milliseconds(),
			})
		})
	}
}