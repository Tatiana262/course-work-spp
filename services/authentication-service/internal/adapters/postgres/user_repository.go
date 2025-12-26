package postgres_adapter

import (
	"authentication-service/internal/contextkeys"
	"authentication-service/internal/core/domain"
	"authentication-service/internal/core/port"
	"context"
	"errors" // Нужен для сравнения ошибок
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository - реализация UserRepositoryPort для PostgreSQL.
type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) (*UserRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("pgxpool.Pool cannot be nil")
	}
	return &UserRepository{
		pool: pool,
	}, nil
}

// Create создает нового пользователя в БД.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "UserRepository",
		"method":    "Create",
		"user_id":   user.ID.String(),
		"email":     user.Email,
	})
	
	query := `INSERT INTO users (id, email, password_hash, role, created_at) VALUES ($1, $2, $3, $4, $5)`
	
	repoLogger.Debug("Executing query to create user.", nil)
	_, err := r.pool.Exec(ctx, query, user.ID, user.Email, user.PasswordHash, user.Role, user.CreatedAt)
	if err != nil {
		repoLogger.Error("Failed to create user", err, port.Fields{"query": query})
		return fmt.Errorf("failed to create user: %w", err)
	}

	repoLogger.Debug("User created successfully.", nil)
	return nil
}

// FindByEmail находит пользователя по email.
// Возвращает (nil, nil), если пользователь не найден.
// Возвращает (nil, error), если произошла ошибка БД.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "UserRepository",
		"method":    "FindByEmail",
		"email":     email,
	})

	query := `SELECT id, email, password_hash, role, created_at FROM users WHERE email = $1`

	repoLogger.Debug("Executing query to find user by email.", nil)
	var user domain.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		// pgx.ErrNoRows - это специальная ошибка, которую возвращает Scan,
		// если запрос не вернул ни одной строки.
		if errors.Is(err, pgx.ErrNoRows) {
			repoLogger.Warn("User not found by email.", nil)
			return nil, nil 
		}
		repoLogger.Error("Failed to find user by email", err, port.Fields{"query": query})
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}

	repoLogger.Debug("User found by email.", port.Fields{"user_id": user.ID.String()})
	return &user, nil
}

// FindByID - аналогично FindByEmail.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "UserRepository",
		"method":    "FindByID",
		"user_id":   id.String(),
	})

	query := `SELECT id, email, password_hash, role, created_at FROM users WHERE id = $1`
	
	repoLogger.Debug("Executing query to find user by ID.", nil)
	var user domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			repoLogger.Warn("User not found by ID.", nil)
			return nil, nil
		}
		repoLogger.Error("Failed to find user by ID", err, port.Fields{"query": query})
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}

	repoLogger.Debug("User found by ID.", nil)
	return &user, nil
}