package port

import (
	"authentication-service/internal/core/domain"
	"context"
	"time"
)



// TokenServicePort определяет, что мы хотим делать с токенами.
type TokenServicePort interface {
	// Генерирует токен для пользователя со сроком жизни.
	GenerateToken(ctx context.Context, user *domain.User, ttl time.Duration) (string, error)
	// Проверяет токен и возвращает "полезную нагрузку" (claims), если он валиден.
	ValidateToken(ctx context.Context, tokenString string) (*domain.Claims, error)
}