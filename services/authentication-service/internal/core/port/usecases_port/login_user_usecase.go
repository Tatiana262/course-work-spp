package usecases_port

import (
	"authentication-service/internal/core/domain"
	"context"
)

type LoginUserUseCasePort interface {
	Execute(ctx context.Context, email, password string) (*domain.User, string, error) // Возвращает JWT токен
}