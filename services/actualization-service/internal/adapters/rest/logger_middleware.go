package rest

import (
	"actualization-service/internal/contextkeys" 
	"actualization-service/internal/core/port"   
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// LoggerMiddleware — это middleware для структурированного логирования
func LoggerMiddleware(logger port.LoggerPort) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := r.Header.Get("X-Trace-ID")
			if traceID == "" {
				traceID = uuid.New().String()
			}

			// Логгер для бизнес-логики (use case, repository)
			coreLogger := logger.WithFields(port.Fields{
				"trace_id": traceID,
			})

			// Логгер для этого middleware
			httpLogger := coreLogger.WithFields(port.Fields{
				"http_method": r.Method,
				"http_path":   r.URL.Path,
				"remote_addr": r.RemoteAddr,
			})

			ctx := r.Context()
			ctx = contextkeys.ContextWithLogger(ctx, coreLogger)
			ctx = contextkeys.ContextWithTraceID(ctx, traceID)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			startTime := time.Now()

			httpLogger.Info("Request started", nil)

			next.ServeHTTP(ww, r.WithContext(ctx))
			
			httpLogger.Info("Request finished", port.Fields{
				"status_code":   ww.Status(),
				"bytes_written": ww.BytesWritten(),
				"duration_ms":   time.Since(startTime).Milliseconds(),
			})
		})
	}
}
