package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool" 
)

// Config хранит конфигурацию для подключения к PostgreSQL
type Config struct {
	DatabaseURL string // "postgres://user:password@host:port/dbname?sslmode=disable"
	// MaxConns int32
	// MinConns int32
	// MaxConnLifetime time.Duration
}

// NewClient создает и возвращает новый пул соединений к PostgreSQL
func NewClient(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL configuration is required")
	}

	// Парсим конфигурацию из URL, если нужно установить доп. параметры пула
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// if cfg.MaxConns > 0 {
	// 	poolConfig.MaxConns = cfg.MaxConns
	// }

	// Подключаемся к базе данных, используя конфигурацию пула
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Проверяем соединение с базой данных
	if err := pool.Ping(ctx); err != nil {
		pool.Close() // Закрываем пул, если пинг не прошел
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return pool, nil
}