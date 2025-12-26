package postgres

import (
	"context"
	"errors"
	"fmt"
	// "log"
	"realt-parser-service/internal/contextkeys"
	"realt-parser-service/internal/core/port"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresLastRunRepository реализует LastRunRepositoryPort для PostgreSQL
type PostgresLastRunRepository struct {
	dbPool *pgxpool.Pool
}

// NewPostgresLastRunRepository создает новый экземпляр PostgresLastRunRepository
func NewPostgresLastRunRepository(dbPool *pgxpool.Pool) (*PostgresLastRunRepository, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("postgres last run repository: dbPool cannot be nil")
	}
	return &PostgresLastRunRepository{dbPool: dbPool}, nil
}

// GetLastRunTimestamp извлекает время последнего запуска для указанного парсера.
func (r *PostgresLastRunRepository) GetLastRunTimestamp(ctx context.Context, parserName string) (time.Time, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "PostgresLastRunRepository",
		"method":    "GetLastRunTimestamp",
	})
	
	var lastRun time.Time
	query := `SELECT last_run_timestamp FROM parser_last_runs WHERE parser_name = $1`

	repoLogger.Debug("Getting last run timestamp", port.Fields{"parser_name": parserName})

	// Выполняем запрос и сканируем результат в переменную lastRun
	err := r.dbPool.QueryRow(ctx, query, parserName).Scan(&lastRun)
	if err != nil {	
		if errors.Is(err, pgx.ErrNoRows) {
			repoLogger.Warn("No last run timestamp found", port.Fields{"parser_name": parserName})
			return time.Time{}, nil
		}
		
		// Если это любая другая ошибка (проблемы с соединением и т.д.)
		repoLogger.Error("Error getting last run timestamp", err, port.Fields{ "parser_name": parserName, })
		return time.Time{}, fmt.Errorf("PostgresLastRunRepo: error querying last run for parser '%s': %w", parserName, err)
	}

	repoLogger.Debug("Found last run timestamp", port.Fields{
		"parser_name":        parserName,
		"last_run_timestamp": lastRun,
	})
	return lastRun, nil
}

// SetLastRunTimestamp устанавливает или обновляет время последнего запуска для указанного парсера.
func (r *PostgresLastRunRepository) SetLastRunTimestamp(ctx context.Context, parserName string, t time.Time) error {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "PostgresLastRunRepository",
		"method":    "SetLastRunTimestamp",
	})

	// Используем ON CONFLICT (UPSERT) — это самый эффективный и атомарный способ.
	query := `
        INSERT INTO parser_last_runs (parser_name, last_run_timestamp)
        VALUES ($1, $2)
        ON CONFLICT (parser_name) DO UPDATE SET last_run_timestamp = EXCLUDED.last_run_timestamp
    `

	repoLogger.Debug("Setting last run timestamp", port.Fields{
		"parser_name":   parserName,
		"new_timestamp": t,
	})

	// Выполняем запрос
	_, err := r.dbPool.Exec(ctx, query, parserName, t)
	if err != nil {
		repoLogger.Error("Error setting last run timestamp", err, port.Fields{ "parser_name": parserName })
		return fmt.Errorf("PostgresLastRunRepo: error setting last run for parser '%s': %w", parserName, err)
	}

	repoLogger.Debug("Successfully set last run timestamp", port.Fields{"parser_name": parserName})
	return nil
}


// CREATE TABLE IF NOT EXISTS parser_last_runs (
//     parser_name VARCHAR(255) PRIMARY KEY,
//     last_run_timestamp TIMESTAMPTZ NOT NULL
// );

// -- Можно добавить индекс для ускорения поиска
// CREATE INDEX IF NOT EXISTS idx_parser_last_runs_parser_name ON parser_last_runs(parser_name);