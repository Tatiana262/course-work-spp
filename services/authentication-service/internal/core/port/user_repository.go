package port

import (
	"context"
	"authentication-service/internal/core/domain"
	"github.com/google/uuid"
)

// UserRepositoryPort определяет, что мы хотим делать с хранилищем пользователей.
type UserRepositoryPort interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}