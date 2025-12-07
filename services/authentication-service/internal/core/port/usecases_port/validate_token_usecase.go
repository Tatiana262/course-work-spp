package usecases_port

import (
	"authentication-service/internal/core/domain"
	"context"
)

type ValidateTokenUseCasePort interface {
	Execute(ctx context.Context, tokenString string) (*domain.Claims, error) // Возвращает claims, если токен валиден
}