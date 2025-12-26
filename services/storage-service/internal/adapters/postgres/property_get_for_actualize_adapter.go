package postgres

import (
	"context"
	"fmt"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"
)

type StatItem struct {
	Category string
	Status   string
	Count    int64
}

func (r *PostgresStorageAdapter) GetActualizationStats(ctx context.Context) ([]domain.StatsByCategory, error) {
	query := `
		SELECT
			category,
			status,
			COUNT(DISTINCT master_object_id) as unique_object_count
		FROM
			general_properties
		GROUP BY
			category, status;
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query object stats: %w", err)
	}
	defer rows.Close()
	
	// Используем map для удобной агрегации
	statsMap := make(map[string]*domain.StatsByCategory)
	
	for rows.Next() {
		var item StatItem
		if err := rows.Scan(&item.Category, &item.Status, &item.Count); err != nil {
			return nil, fmt.Errorf("failed to scan stats item: %w", err)
		}
		
		// Если мы еще не видели эту категорию, создаем для нее запись в map
		if _, ok := statsMap[item.Category]; !ok {
			statsMap[item.Category] = &domain.StatsByCategory{
				SystemName:  item.Category,
				DisplayName: translateCategory(item.Category), // Используем ваш переводчик
			}
		}
		
		// Заполняем счетчики
		if item.Status == "active" {
			statsMap[item.Category].ActiveCount = item.Count
		} else if item.Status == "archived" {
			statsMap[item.Category].ArchivedCount = item.Count
		}
	}
    if err := rows.Err(); err != nil {
        return nil, err
    }

	// Преобразуем map в срез для JSON-ответа
	result := make([]domain.StatsByCategory, 0, len(statsMap))
	for _, stat := range statsMap {
		result = append(result, *stat)
	}
    
	return result, nil
}

func (a *PostgresStorageAdapter) GetActiveIDsForActualization(ctx context.Context, category string, limit int) ([]domain.PropertyBasicInfo, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "PostgresStorageAdapter",
		"method":    "GetActiveIDsForActualization",
		"category":  category,
		"limit":     limit,
	})
	
    // Здесь будет ваш SQL-запрос. Например, выбрать самые старые активные объекты.
    // query := `SELECT id, ad_link, source_ad_id, source, updated_at FROM general_properties 
    //            WHERE category = $1 AND status = 'active' ORDER BY updated_at ASC LIMIT $2`

	query := `WITH oldest_master_objects AS (
				SELECT master_object_id
				FROM general_properties
				WHERE
					category = $1
					AND status = 'active'
				GROUP BY master_object_id
				ORDER BY MIN(updated_at) ASC
				LIMIT $2 -- limit
			)

			SELECT id, ad_link, source_ad_id, source, updated_at
			FROM general_properties
			WHERE
				master_object_id IN (SELECT master_object_id FROM oldest_master_objects)
			AND status = 'active';`
    
	repoLogger.Debug("Querying for active objects to actualize.", nil)
	rows, err := a.pool.Query(ctx, query, category, limit)

	if err != nil {
		repoLogger.Error("Failed to query active objects", err, port.Fields{"query": query})
		return nil, fmt.Errorf("PostgresStorageAdapter: failed to query active objects: %w", err)
	}
	defer rows.Close()


	var objectsInfo []domain.PropertyBasicInfo
	for rows.Next() {
		var inf domain.PropertyBasicInfo
		
		if err := rows.Scan(&inf.ID, &inf.Link, &inf.AdID, &inf.Source, &inf.UpdatedAt); err != nil {
			repoLogger.Error("Failed to scan active object row", err, nil)
			return nil, fmt.Errorf("PostgresStorageAdapter: failed to scan active object: %w", err)
		}
		
		objectsInfo = append(objectsInfo, inf)
	}

	// Выведите, что именно было отсканировано
    // log.Println(objectsInfo)
	
	// Проверяем на ошибки, которые могли возникнуть во время итерации
	if err = rows.Err(); err != nil {
		repoLogger.Error("Error during active rows iteration", err, nil)
        return nil, fmt.Errorf("PostgresStorageAdapter: error during active rows iteration: %w", err)
    }

	if len(objectsInfo) == 0 {
		// (Опционально) Явный лог, что ничего не найдено. Можно использовать уровень Warn или Info.
		repoLogger.Info("Query successful, but no active objects found to actualize.", nil)
	} else {
		repoLogger.Info("Successfully found active objects", port.Fields{"count": len(objectsInfo)})
	}
    return objectsInfo, nil 
}


func (a *PostgresStorageAdapter) GetArchivedIDsForActualization(ctx context.Context, category string, limit int) ([]domain.PropertyBasicInfo, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "PostgresStorageAdapter",
		"method":    "GetArchivedIDsForActualization",
		"category":  category,
		"limit":     limit,
	})

    // Здесь будет ваш SQL-запрос. Например, выбрать самые старые активные объекты.
    // query := `SELECT id, ad_link, source_ad_id, source, updated_at FROM general_properties 
    //            WHERE category = $1 AND status = 'archived' ORDER BY updated_at ASC LIMIT $2`

	query := `WITH oldest_master_objects AS (
				SELECT master_object_id
				FROM general_properties
				WHERE
					category = $1
					AND status = 'archived'
				GROUP BY master_object_id
				ORDER BY MIN(updated_at) ASC
				LIMIT $2 -- limit
			)

			SELECT id, ad_link, source_ad_id, source, updated_at
			FROM general_properties
			WHERE
				master_object_id IN (SELECT master_object_id FROM oldest_master_objects)
			AND status = 'archived';`
    
	repoLogger.Debug("Querying for archived objects to actualize.", nil)
	rows, err := a.pool.Query(ctx, query, category, limit)

	if err != nil {
		repoLogger.Error("Failed to query archived objects", err, port.Fields{"query": query})
		return nil, fmt.Errorf("PostgresStorageAdapter: failed to query archived objects: %w", err)
	}
	defer rows.Close()


	var objectsInfo []domain.PropertyBasicInfo
	for rows.Next() {
		var inf domain.PropertyBasicInfo
		
		if err := rows.Scan(&inf.ID, &inf.Link, &inf.AdID, &inf.Source, &inf.UpdatedAt); err != nil {
			repoLogger.Error("Failed to scan archived object row", err, nil)
			return nil, fmt.Errorf("PostgresStorageAdapter: failed to scan archived object: %w", err)
		}
		
		objectsInfo = append(objectsInfo, inf)
	}
	
	 // Проверяем на ошибки, которые могли возникнуть во время итерации
	if err = rows.Err(); err != nil {
		repoLogger.Error("Error during archived rows iteration", err, nil)
        return nil, fmt.Errorf("PostgresStorageAdapter: error during archived rows iteration: %w", err)
    }

	if len(objectsInfo) == 0 {
		// (Опционально) Явный лог, что ничего не найдено. Можно использовать уровень Warn или Info.
		repoLogger.Info("Query successful, but no active objects found to actualize.", nil)
	} else {
		repoLogger.Info("Successfully found active objects", port.Fields{"count": len(objectsInfo)})
	}
    return objectsInfo, nil 
}


func (a *PostgresStorageAdapter) GetObjectsByIDForActualization(ctx context.Context, masterObjectID string) ([]domain.PropertyBasicInfo, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "PostgresStorageAdapter",
		"method":    "GetObjectByIDForActualization",
		"master_object_id":  masterObjectID,
	})

	// Здесь будет ваш SQL-запрос. Например, выбрать самые старые активные объекты.
    query := `
		SELECT id, ad_link, source_ad_id, source, updated_at 
        FROM general_properties 
        WHERE master_object_id = $1 AND status = 'active'
		ORDER BY updated_at DESC
	`
    
	rows, err := a.pool.Query(ctx, query, masterObjectID)
    if err != nil {
        repoLogger.Error("Failed to query objects by master_id", err, nil)
        return nil, fmt.Errorf("failed to find objects by master_id %s: %w", masterObjectID, err)
    }
    defer rows.Close()

	var objectsInfo []domain.PropertyBasicInfo
    for rows.Next() {
        var info domain.PropertyBasicInfo
        if err := rows.Scan(&info.ID, &info.Link, &info.AdID, &info.Source, &info.UpdatedAt); err != nil {
            return nil, fmt.Errorf("failed to scan objects by master_id: %w", err)
        }
        objectsInfo = append(objectsInfo, info)
    }

    if err := rows.Err(); err != nil {
        return nil, err
    }

    repoLogger.Info("Successfully found objects by master_id", port.Fields{"count": len(objectsInfo)})
    return objectsInfo, nil
}

// func (a *PostgresStorageAdapter) GetObjectByIDForActualization(ctx context.Context, id string) (*domain.PropertyBasicInfo, error) {
// 	logger := contextkeys.LoggerFromContext(ctx)
// 	repoLogger := logger.WithFields(port.Fields{
// 		"component": "PostgresStorageAdapter",
// 		"method":    "GetObjectByIDForActualization",
// 		"id":  id,
// 	})

// 	// Здесь будет ваш SQL-запрос. Например, выбрать самые старые активные объекты.
//     query := `SELECT id, ad_link, source_ad_id, source, updated_at FROM general_properties 
//                WHERE id = $1`
    
// 	var objectInfo domain.PropertyBasicInfo
// 	err := a.pool.QueryRow(ctx, query, id).Scan(
// 		&objectInfo.ID, 
// 		&objectInfo.Link, 
// 		&objectInfo.AdID, 
// 		&objectInfo.Source, 
// 		&objectInfo.UpdatedAt,
// 	);
// 	if err != nil {
// 		// if errors.Is(err, pgx.ErrNoRows) {
// 		// 	return nil, nil 
// 		// }
// 		repoLogger.Error("Failed to query object", err, port.Fields{"query": query})
// 		return nil, fmt.Errorf("PostgresStorageAdapter: failed to find object with id %s: %w", id, err)
// 	}

// 	repoLogger.Info("Successfully found object", nil)

//     return &objectInfo, nil
// }