package usecases_port

import (
	"authentication-service/internal/core/domain"
	"context"
)

type RegisterUserUseCasePort interface {
	Execute(ctx context.Context, email, password string) (*domain.User, string, error)  // Возвращает ID нового пользователя
}