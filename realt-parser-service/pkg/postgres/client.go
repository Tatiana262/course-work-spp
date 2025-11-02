package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool" 
)

// хранит конфигурацию для подключения к PostgreSQL
type Config struct {
	DatabaseURL string
}

// создает и возвращает новый пул соединений к PostgreSQL
func NewClient(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL configuration is required")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close() 
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return pool, nil
}