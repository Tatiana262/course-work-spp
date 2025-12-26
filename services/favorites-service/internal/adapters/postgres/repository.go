package postgres_adapter

import (
	"context"
	"errors"
	"favorites-service/internal/contextkeys"
	"favorites-service/internal/core/domain"
	"favorites-service/internal/core/port"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresFavoritesRepository - реализация порта для PostgreSQL.
type PostgresFavoritesRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresFavoritesRepository - конструктор.
func NewPostgresFavoritesRepository(pool *pgxpool.Pool) (*PostgresFavoritesRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("pgxpool.Pool cannot be nil")
	}
	return &PostgresFavoritesRepository{pool: pool}, nil
}

// Add добавляет запись в user_favorites.
func (r *PostgresFavoritesRepository) Add(ctx context.Context, userID, masterObjectID uuid.UUID) error {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component":        "PostgresFavoritesRepository",
		"method":           "Add",
		"user_id":          userID,
		"master_object_id": masterObjectID,
	})
	
	repoLogger.Debug("Attempting to add to favorites.", nil)
	query := `INSERT INTO user_favorites (user_id, master_object_id) VALUES ($1, $2)`

	_, err := r.pool.Exec(ctx, query, userID, masterObjectID)
	if err != nil {
		// Проверяем на ошибку нарушения unique constraint (PRIMARY KEY).
		// Это означает, что такая запись уже существует. В данном случае это не ошибка.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // 23505 - unique_violation
			repoLogger.Warn("Favorite already exists, operation considered successful.", nil)
			return nil // Запись уже существует, считаем операцию успешной.
		}
		repoLogger.Error("Failed to add favorite", err, port.Fields{"query": query})
		return fmt.Errorf("failed to add favorite: %w", err)
	}

	repoLogger.Debug("Successfully added to favorites.", nil)
	return nil
}

// Remove удаляет запись из user_favorites.
func (r *PostgresFavoritesRepository) Remove(ctx context.Context, userID, masterObjectID uuid.UUID) error {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component":        "PostgresFavoritesRepository",
		"method":           "Remove",
		"user_id":          userID,
		"master_object_id": masterObjectID,
	})

	repoLogger.Debug("Attempting to remove from favorites.", nil)
	query := `DELETE FROM user_favorites WHERE user_id = $1 AND master_object_id = $2`

	// Exec возвращает CommandTag, который можно проверить, чтобы узнать, была ли удалена строка.
	// Но для простоты можно просто проверить на ошибку.
	cmdTag, err := r.pool.Exec(ctx, query, userID, masterObjectID)
	if err != nil {
		repoLogger.Error("Failed to remove favorite", err, port.Fields{"query": query})
		return fmt.Errorf("failed to remove favorite: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		repoLogger.Warn("Attempted to remove a favorite that did not exist.", nil)
	} else {
		repoLogger.Debug("Successfully removed from favorites.", nil)
	}
	return nil
}

func (r *PostgresFavoritesRepository) FindFavoritesIdsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "PostgresFavoritesRepository",
		"method":    "FindFavoritesIdsByUser",
		"user_id":   userID,
	})

	dataQuery := "SELECT master_object_id FROM user_favorites WHERE user_id = $1"
	rows, err := r.pool.Query(ctx, dataQuery, userID)
	if err != nil {
		repoLogger.Error("Failed to query favorite IDs", err, port.Fields{"query": dataQuery})
		return nil, fmt.Errorf("failed to query favorite IDs: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			repoLogger.Error("Failed to scan favorite ID row", err, nil)
			return nil, fmt.Errorf("failed to scan favorite ID: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		repoLogger.Error("Error during favorite IDs iteration", err, nil)
		return nil, fmt.Errorf("error during favorite IDs iteration: %w", err)
	}

	return ids, nil
}

// FindPaginatedByUser находит ID избранных объектов с пагинацией.
func (r *PostgresFavoritesRepository) FindPaginatedByUser(ctx context.Context, userID uuid.UUID, limit, offset int) (*domain.PaginatedFavoriteIDs, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "PostgresFavoritesRepository",
		"method":    "FindPaginatedByUser",
		"user_id":   userID,
		"limit":     limit,
		"offset":    offset,
	})

	repoLogger.Debug("Starting transaction to find paginated favorites.", nil)
	// Выполняем два запроса в одной "виртуальной" транзакции для консистентности.
	// Использование `tx.Begin` здесь не обязательно, но это хорошая практика.
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		repoLogger.Error("Failed to begin transaction", err, nil)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Запрос на общее количество
	var totalCount int64
	countQuery := "SELECT COUNT(*) FROM user_favorites WHERE user_id = $1"
	if err := tx.QueryRow(ctx, countQuery, userID).Scan(&totalCount); err != nil {
		repoLogger.Error("Failed to count favorites", err, port.Fields{"query": countQuery})
		return nil, fmt.Errorf("failed to count favorites: %w", err)
	}
	repoLogger.Debug("Total favorites for user", port.Fields{"total_count": totalCount})

	// Если избранных нет, сразу возвращаем результат
	if totalCount == 0 {
		return &domain.PaginatedFavoriteIDs{
			MasterObjectIDs: []uuid.UUID{},
			TotalCount:      0,
			CurrentPage:  offset/limit + 1, // Показываем, на какой странице мы находимся
			ItemsPerPage: limit,               // И с какими параметрами
		}, nil
	}

	// 2. Запрос на получение ID для текущей страницы
	// Сортируем по `created_at DESC`, чтобы новые были первыми.
	dataQuery := "SELECT master_object_id FROM user_favorites WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3"
	rows, err := tx.Query(ctx, dataQuery, userID, limit, offset)
	if err != nil {
		repoLogger.Error("Failed to query favorite IDs", err, port.Fields{"query": dataQuery})
		return nil, fmt.Errorf("failed to query favorite IDs: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0, limit)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			repoLogger.Error("Failed to scan favorite ID row", err, nil)
			return nil, fmt.Errorf("failed to scan favorite ID: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		repoLogger.Error("Error during favorite IDs iteration", err, nil)
		return nil, fmt.Errorf("error during favorite IDs iteration: %w", err)
	}

	// Если все прошло успешно, коммитим транзакцию
	if err := tx.Commit(ctx); err != nil {
		repoLogger.Error("Failed to commit transaction", err, nil)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	repoLogger.Debug("Successfully found paginated favorites.", port.Fields{"found_on_page": len(ids)})
	return &domain.PaginatedFavoriteIDs{
		MasterObjectIDs: ids,
		TotalCount:      totalCount,
		CurrentPage:  offset/limit + 1, // Показываем, на какой странице мы находимся
		ItemsPerPage: limit,               // И с какими параметрами
	}, nil
}