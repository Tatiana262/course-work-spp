package postgres_adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"task-service/internal/contextkeys"
	"task-service/internal/core/domain" // Убедитесь, что путь верный
	"task-service/internal/core/port"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresTaskRepository - реализация порта для PostgreSQL.
type PostgresTaskRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresTaskRepository - конструктор.
func NewPostgresTaskRepository(pool *pgxpool.Pool) (*PostgresTaskRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("pgxpool.Pool cannot be nil")
	}
	return &PostgresTaskRepository{pool: pool}, nil
}

// Create создает новую задачу в БД.
func (r *PostgresTaskRepository) Create(ctx context.Context, task *domain.Task) error {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "PostgresTaskRepository",
		"method":    "Create",
		"task_id":   task.ID.String(),
		"task": *task,
	})
	
	repoLogger.Debug("Creating new task in DB", nil)

	query := `
		INSERT INTO tasks (id, name, type, status, created_at, created_by_user_id, result_summary)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	// Инициализируем result_summary пустым JSON-объектом '{}'
	summaryJSON, _ := json.Marshal(task.ResultSummary)

	_, err := r.pool.Exec(ctx, query,
		task.ID,
		task.Name,
		task.Type,
		task.Status,
		task.CreatedAt,
		task.CreatedByUserID,
		summaryJSON,
	)
	if err != nil {
		repoLogger.Error("Failed to create task", err, port.Fields{"query": query})
		return fmt.Errorf("failed to create task: %w", err)
	}

	repoLogger.Debug("Task created successfully", nil)
	return nil
}

// Update обновляет существующую задачу.
func (r *PostgresTaskRepository) Update(ctx context.Context, task *domain.Task) error {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "PostgresTaskRepository",
		"method":    "Update",
		"task_id":   task.ID.String(),
	})

	repoLogger.Debug("Updating task in DB", port.Fields{"new_status": task.Status})

	// summaryJSON, err := json.Marshal(task.ResultSummary)
	// if err != nil {
	// 	repoLogger.Error("Failed to marshal result summary for update", err, nil)
	// 	return fmt.Errorf("failed to marshal result summary: %w", err)
	// }

	query := `
		UPDATE tasks
		SET
			name = $2,
			type = $3,
			status = $4,
			started_at = $5,
			finished_at = $6
		WHERE id = $1
	`
	cmdTag, err := r.pool.Exec(ctx, query,
		task.ID,
		task.Name,
		task.Type,
		task.Status,
		task.StartedAt,
		task.FinishedAt,
	)
	if err != nil {
		repoLogger.Error("Failed to update task", err, port.Fields{"query": query})
		return fmt.Errorf("failed to update task: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		repoLogger.Warn("Update failed: task not found", nil)
		return domain.ErrTaskNotFound
	}

	repoLogger.Debug("Task updated successfully", nil)
	return nil
}

// FindByID находит одну задачу по ее ID.
func (r *PostgresTaskRepository) FindByID(ctx context.Context, taskID uuid.UUID) (*domain.Task, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "PostgresTaskRepository",
		"method":    "FindByID",
		"task_id":   taskID.String(),
	})

	repoLogger.Debug("Finding task by ID.", nil)

	query := `
		SELECT id, name, type, status, result_summary, created_at, started_at, finished_at, created_by_user_id
		FROM tasks
		WHERE id = $1
	`
	var task domain.Task
	var summaryJSON []byte

	err := r.pool.QueryRow(ctx, query, taskID).Scan(
		&task.ID,
		&task.Name,
		&task.Type,
		&task.Status,
		&summaryJSON,
		&task.CreatedAt,
		&task.StartedAt,
		&task.FinishedAt,
		&task.CreatedByUserID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			repoLogger.Warn("Task not found.", nil)
			return nil, domain.ErrTaskNotFound
		}
		repoLogger.Error("Failed to find task by ID", err, port.Fields{"query": query})
		return nil, fmt.Errorf("failed to find task by id: %w", err)
	}

	if err := json.Unmarshal(summaryJSON, &task.ResultSummary); err != nil {
		repoLogger.Error("Failed to unmarshal result summary from DB", err, nil)
		return nil, fmt.Errorf("failed to unmarshal result summary: %w", err)
	}

	repoLogger.Debug("Task found successfully.", nil)
	return &task, nil
}


// FindAll находит все задачи для конкретного пользователя с пагинацией
func (r *PostgresTaskRepository) FindAll(ctx context.Context, createdByUserID uuid.UUID, limit, offset int) ([]domain.Task, int64, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "PostgresTaskRepository",
		"method":    "FindAll",
		"user_id":   createdByUserID.String(),
		"limit":	limit,
		"offset":   offset,
	})

	repoLogger.Debug("Starting transaction to find all tasks for user.", nil)
    tx, err := r.pool.Begin(ctx)
    if err != nil {
		repoLogger.Error("Failed to begin transaction", err, nil)
        return nil, 0, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback(ctx)

    // 1. Запрос на общее количество С УЧЕТОМ ПОЛЬЗОВАТЕЛЯ
    var totalCount int64
    countQuery := "SELECT COUNT(*) FROM tasks WHERE created_by_user_id = $1"
    if err := tx.QueryRow(ctx, countQuery, createdByUserID).Scan(&totalCount); err != nil {
		repoLogger.Error("Failed to count tasks", err, port.Fields{"query": countQuery})
        return nil, 0, fmt.Errorf("failed to count tasks for user %s: %w", createdByUserID, err)
    }

    if totalCount == 0 {
        return []domain.Task{}, 0, nil
    }

    // 2. Запрос на получение данных С УЧЕТОМ ПОЛЬЗОВАТЕЛЯ
    dataQuery := `
        SELECT id, name, type, status, result_summary, created_at, started_at, finished_at, created_by_user_id
        FROM tasks
        WHERE created_by_user_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `
    rows, err := tx.Query(ctx, dataQuery, createdByUserID, limit, offset)
    if err != nil {
		repoLogger.Error("Failed to query tasks", err, port.Fields{"query": dataQuery})
        return nil, 0, fmt.Errorf("failed to query tasks for user %s: %w", createdByUserID, err)
    }
    defer rows.Close()

    tasks := make([]domain.Task, 0, limit)
    for rows.Next() {
        var task domain.Task
        var summaryJSON []byte
        if err := rows.Scan(
            &task.ID, &task.Name, &task.Type, &task.Status, &summaryJSON, &task.CreatedAt,
            &task.StartedAt, &task.FinishedAt, &task.CreatedByUserID,
        ); err != nil {
			repoLogger.Error("Failed to scan task row", err, nil)
            return nil, 0, fmt.Errorf("failed to scan task: %w", err)
        }
        if err := json.Unmarshal(summaryJSON, &task.ResultSummary); err != nil {
			repoLogger.Error("Failed to unmarshal task summary", err, nil)
            return nil, 0, fmt.Errorf("failed to unmarshal task summary: %w", err)
        }
        tasks = append(tasks, task)
    }
    
    if err := rows.Err(); err != nil {
		repoLogger.Error("Error during tasks iteration", err, nil)
        return nil, 0, fmt.Errorf("error during tasks iteration: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
		repoLogger.Error("Failed to commit transaction", err, nil)
        return nil, 0, fmt.Errorf("failed to commit transaction: %w", err)
    }

    return tasks, totalCount, nil
}

// IncrementSummary атомарно обновляет числовые значения в JSONB поле result_summary.
// func (r *PostgresTaskRepository) IncrementSummary(ctx context.Context, taskID uuid.UUID, results map[string]int) error {
// 	logger := contextkeys.LoggerFromContext(ctx)
// 	repoLogger := logger.WithFields(port.Fields{
// 		"component": "PostgresTaskRepository",
// 		"method":    "IncrementSummary",
// 		"task_id":   taskID.String(),
// 	})

// 	repoLogger.Debug("Starting transaction to increment summary", port.Fields{"updates": results})
// 	// Этот метод будет выполнять несколько UPDATE операций в одной транзакции
// 	// для каждого ключа в `results`.
// 	tx, err := r.pool.Begin(ctx)
// 	if err != nil {
// 		repoLogger.Error("Failed to begin transaction", err, nil)
// 		return fmt.Errorf("failed to begin transaction for incrementing summary: %w", err)
// 	}
// 	defer tx.Rollback(ctx)

// 	for key, incrementValue := range results {
// 		// if incrementValue == 0 {
// 		// 	continue // Пропускаем нулевые инкременты
// 		// }

// 		// Этот SQL - магия работы с JSONB в PostgreSQL
// 		query := `
// 			UPDATE tasks
// 			SET result_summary = jsonb_set(
// 				result_summary,
// 				'{%s}', -- Путь к ключу, например, '{"processed_count"}'
// 				(COALESCE(result_summary->>'%s', '0')::int + %d)::text::jsonb
// 			)
// 			WHERE id = '%s'
// 		`
// 		// Формируем безопасный запрос
// 		// ВАЖНО: Мы используем Sprintf здесь, потому что имена ключей (`key`) не могут быть параметризованы.
// 		// Мы доверяем этим ключам, так как они приходят из нашей же системы.
// 		// `incrementValue` и `taskID` также можно было бы передать как параметры, но для простоты
// 		// и из-за динамического построения, используем Sprintf.
// 		formattedQuery := fmt.Sprintf(query, key, key, incrementValue, taskID)

// 		repoLogger.Debug("Executing summary increment", port.Fields{"key": key, "value": incrementValue})

// 		_, err := tx.Exec(ctx, formattedQuery)
// 		if err != nil {
// 			repoLogger.Error("Failed to increment summary key", err, port.Fields{"key": key, "query": query})
// 			return fmt.Errorf("failed to increment summary key '%s' for task %s: %w", key, taskID, err)
// 		}
// 	}

// 	repoLogger.Debug("Committing summary increments.", nil)
// 	return tx.Commit(ctx)
// }


func (r *PostgresTaskRepository) IncrementSummary(ctx context.Context, taskID uuid.UUID, results map[string]int) (*domain.Task, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "PostgresTaskRepository",
		"method":    "IncrementSummary",
		"task_id":   taskID.String(),
	})

	if len(results) == 0 {
		return r.FindByID(ctx, taskID)
	}

	resultsJSON, err := json.Marshal(results)
	if err != nil {
		repoLogger.Error("Failed to marshal task result", err, nil)
		return nil, fmt.Errorf("failed to marshal results for increment: %w", err)
	}

    // ---> УПРОЩЕННЫЙ, НО АТОМАРНЫЙ SQL <---
	query := `
        WITH
        -- 1. Превращаем JSON с инкрементами в таблицу (ключ, значение)
        increments(key, value) AS (
            SELECT key, value::bigint FROM jsonb_each_text($2::jsonb)
        ),
        -- 2. Загружаем и блокируем текущую задачу
        current_task AS (
            SELECT * FROM tasks WHERE id = $1 FOR UPDATE
        ),
        -- 3. Вычисляем новый JSONB
        new_summary AS (
            SELECT
                (
                    SELECT COALESCE(result_summary, '{}'::jsonb) FROM current_task
                ) || (
                    SELECT jsonb_object_agg(
                        inc.key,
                        COALESCE((SELECT result_summary FROM current_task)->>inc.key, '0')::bigint + inc.value
                    )
                    FROM increments inc
                ) AS final_summary
        )
        -- 4. Обновляем и возвращаем
        UPDATE tasks
        SET result_summary = (SELECT final_summary FROM new_summary)
        WHERE id = $1
        RETURNING id, name, type, status, result_summary, created_at, started_at, finished_at, created_by_user_id;
    `
    
    var task domain.Task
	var summaryJSON []byte

	err = r.pool.QueryRow(ctx, query, taskID, resultsJSON).Scan(
		&task.ID, &task.Name, &task.Type, &task.Status, &summaryJSON,
		&task.CreatedAt, &task.StartedAt, &task.FinishedAt, &task.CreatedByUserID,
	)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
			repoLogger.Warn("Task not found.", nil)
            return nil, domain.ErrTaskNotFound
        }
		repoLogger.Error("failed to increment summary and return task", err, nil)
        return nil, fmt.Errorf("failed to increment summary and return task: %w", err)
    }
    
    if err := json.Unmarshal(summaryJSON, &task.ResultSummary); err != nil {
		repoLogger.Error("failed to unmarshal updated result summary", err, nil)
        return nil, fmt.Errorf("failed to unmarshal updated result summary: %w", err)
    }

	return &task, nil
}