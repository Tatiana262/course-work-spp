package postgres

import (
	"context"
	"fmt"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"
)


func (a *PostgresStorageAdapter) GetActiveIDsForActualization(ctx context.Context, category string, limit int) ([]domain.PropertyBasicInfo, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "PostgresStorageAdapter",
		"method":    "GetActiveIDsForActualization",
		"category":  category,
		"limit":     limit,
	})
	
    // Здесь будет ваш SQL-запрос. Например, выбрать самые старые активные объекты.
    query := `SELECT id, ad_link, source_ad_id, source, updated_at FROM general_properties 
               WHERE category = $1 AND status = 'active' ORDER BY updated_at ASC LIMIT $2`
    
	repoLogger.Info("Querying for active objects to actualize.", nil)
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
    query := `SELECT id, ad_link, source_ad_id, source, updated_at FROM general_properties 
               WHERE category = $1 AND status = 'archived' ORDER BY updated_at ASC LIMIT $2`
    
	repoLogger.Info("Querying for archived objects to actualize.", nil)
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

func (a *PostgresStorageAdapter) GetObjectByIDForActualization(ctx context.Context, id string) (*domain.PropertyBasicInfo, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "PostgresStorageAdapter",
		"method":    "GetObjectByIDForActualization",
		"id":  id,
	})

	// Здесь будет ваш SQL-запрос. Например, выбрать самые старые активные объекты.
    query := `SELECT id, ad_link, source_ad_id, source, updated_at FROM general_properties 
               WHERE id = $1`
    
	var objectInfo domain.PropertyBasicInfo
	err := a.pool.QueryRow(ctx, query, id).Scan(
		&objectInfo.ID, 
		&objectInfo.Link, 
		&objectInfo.AdID, 
		&objectInfo.Source, 
		&objectInfo.UpdatedAt,
	);
	if err != nil {
		// if errors.Is(err, pgx.ErrNoRows) {
		// 	return nil, nil 
		// }
		repoLogger.Error("Failed to query object", err, port.Fields{"query": query})
		return nil, fmt.Errorf("PostgresStorageAdapter: failed to find object with id %s: %w", id, err)
	}

	repoLogger.Info("Successfully found object", nil)

	// defer rows.Close()

	// if rows.Next() {				
	// 	if err := rows.Scan(&objectInfo.ID, &objectInfo.Link, &objectInfo.AdID, &objectInfo.Source, &objectInfo.UpdatedAt); err != nil {
	// 		return nil, fmt.Errorf("PostgresStorageAdapter: failed to scan object with id %s: %w", id, err)
	// 	}		
	// }
	
	//  // Проверяем на ошибки, которые могли возникнуть во время итерации
	// if err = rows.Err(); err != nil {
    //     return nil, fmt.Errorf("PostgresStorageAdapter: error during rows iteration for object id = %s: %w", id, err)
    // }

    return &objectInfo, nil
}