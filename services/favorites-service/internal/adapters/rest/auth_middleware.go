package rest

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// Определяем кастомный тип для ключа контекста, чтобы избежать коллизий.
type contextKey string
const userIDKey = contextKey("userID")

// AuthMiddleware - middleware для извлечения userID из заголовка.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIDStr := r.Header.Get("X-User-ID")
		if userIDStr == "" {
			WriteJSONError(w, http.StatusUnauthorized, "X-User-ID header is missing")
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			WriteJSONError(w, http.StatusUnauthorized, "Invalid X-User-ID header format")
			return
		}

		// Добавляем userID в контекст запроса
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		
		// Передаем управление следующему обработчику в цепочке с новым контекстом
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}