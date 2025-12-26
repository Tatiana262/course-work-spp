package server

import (
	"api-gateway/internal/auth"
	"context"
	// "log"
	"net/http"
	"strings"
)

type AuthMiddleware struct {
	authClient *auth.Client
}

func NewAuthMiddleware(authClient *auth.Client) *AuthMiddleware {
	return &AuthMiddleware{authClient: authClient}
}

// Authenticate - middleware для проверки JWT
func (am *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем токен из заголовка Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}
		
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, "Invalid token format", http.StatusUnauthorized)
			return
		}

		// Валидируем токен, делая запрос к auth-service
		claims, err := am.authClient.ValidateToken(r.Context(), tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		
		// Добавляем информацию о пользователе в контекст запроса,
		// чтобы следующие middleware и прокси могли ее использовать
		ctx := context.WithValue(r.Context(), "user_claims", claims)
		
		// 4. Модифицируем запрос, добавляя заголовок для внутренних сервисов
		r.Header.Set("X-User-ID", claims.UserID)
		
		// Передаем управление дальше, но с новым контекстом
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole - middleware для проверки роли пользователя
func (am *AuthMiddleware) RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Извлекаем claims, которые были добавлены middleware Authenticate
			claims, ok := r.Context().Value("user_claims").(*auth.Claims)
			if !ok {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			
			// Проверяем роль
			if claims.Role != requiredRole {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			
			// Если роль подходит, передаем управление дальше
			next.ServeHTTP(w, r)
		})
	}
}