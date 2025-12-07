package rest

import (
	"storage-service/internal/contextkeys" // Путь к вашим утилитам контекста
	"storage-service/internal/core/port"   // Путь к порту логгера
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// LoggerMiddleware — это наш middleware для структурированного логирования.
func LoggerMiddleware(logger port.LoggerPort) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := r.Header.Get("X-Trace-ID")
			if _, err := uuid.Parse(traceID); err != nil {
				traceID = uuid.New().String()
			}

			// --- Элегантное разделение ---

			// 1. Логгер ТОЛЬКО для бизнес-логики (use case, repository).
			// Содержит только то, что важно для всех слоев.
			coreLogger := logger.WithFields(port.Fields{
				"trace_id": traceID,
			})

			// 2. Логгер ТОЛЬКО для этого middleware.
			// Он "наследует" coreLogger и добавляет HTTP-специфичные поля.
			httpLogger := coreLogger.WithFields(port.Fields{
				"http_method": r.Method,
				"http_path":   r.URL.Path,
				"remote_addr": r.RemoteAddr,
			})
			
			// 3. В контекст для use case'ов кладем "чистый" логгер.
			ctx := r.Context()
			ctx = contextkeys.ContextWithLogger(ctx, coreLogger) // <-- КЛАДЕМ ЧИСТЫЙ ЛОГГЕР
			ctx = contextkeys.ContextWithTraceID(ctx, traceID)
			
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			startTime := time.Now()
			
			// Используем HTTP-логгер для логов самого middleware
			httpLogger.Info("Request started", nil)
			
			next.ServeHTTP(ww, r.WithContext(ctx))
			
			// И снова используем HTTP-логгер для финального лога
			httpLogger.Info("Request finished", port.Fields{
				"status_code": ww.Status(),
				"bytes_written": ww.BytesWritten(),
				"duration_ms":   time.Since(startTime).Milliseconds(),
			})
		})
	}
}