package rest

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// Определяем кастомный тип для ключа контекста
type contextKey string
const userIDKey = contextKey("userID")

// AuthMiddleware - middleware для извлечения userID из заголовка X-User-ID.
// Этот заголовок должен добавляться API Gateway после валидации JWT
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIDStr := r.Header.Get("X-User-ID")
		if userIDStr == "" {
			
			// Ошибка указывает либо на проблему конфигурации, либо на прямой доступ в обход Gateway
			WriteJSONError(w, http.StatusUnauthorized, "Authentication error: User ID header is missing")
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			WriteJSONError(w, http.StatusUnauthorized, "Authentication error: Invalid User ID format")
			return
		}

		// Добавляем userID в контекст запроса, чтобы хендлеры могли его использовать
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		
		// Передаем управление следующему обработчику в цепочке с обогащенным контекстом
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}