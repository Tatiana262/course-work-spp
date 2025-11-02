package postgres

import (
	"context"
	"fmt"
	"storage-service/internal/core/domain"
)


func (a *PostgresStorageAdapter) GetActiveIDsForActualization(ctx context.Context, limit, offset int) ([]domain.PropertyBasicInfo, error) {
    query := `SELECT id, source_ad_id, source, updated_at FROM general_properties 
               WHERE is_active = true ORDER BY updated_at ASC LIMIT $1 OFFSET $2`
    
	rows, err := a.pool.Query(ctx, query, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("PostgresStorageAdapter: failed to query active objects: %w", err)
	}
	defer rows.Close()


	var objectsInfo []domain.PropertyBasicInfo
	for rows.Next() {
		var inf domain.PropertyBasicInfo
		
		if err := rows.Scan(&inf.ID, &inf.SourceAdID, &inf.Source, &inf.UpdatedAt); err != nil {
			return nil, fmt.Errorf("PostgresStorageAdapter: failed to scan active object: %w", err)
		}
		
		objectsInfo = append(objectsInfo, inf)
	}
	
	if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("PostgresStorageAdapter: error during active rows iteration: %w", err)
    }

    return objectsInfo, nil 
}


func (a *PostgresStorageAdapter) GetArchivedIDsForActualization(ctx context.Context, limit, offset int) ([]domain.PropertyBasicInfo, error) {
    query := `SELECT id, source_ad_id, source, updated_at FROM general_properties 
               WHERE is_active = false ORDER BY updated_at ASC LIMIT $1 OFFSET $2`
    
	rows, err := a.pool.Query(ctx, query, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("PostgresStorageAdapter: failed to query archived objects: %w", err)
	}
	defer rows.Close()


	var objectsInfo []domain.PropertyBasicInfo
	for rows.Next() {
		var inf domain.PropertyBasicInfo
		
		if err := rows.Scan(&inf.ID, &inf.SourceAdID, &inf.Source, &inf.UpdatedAt); err != nil {
			return nil, fmt.Errorf("PostgresStorageAdapter: failed to scan archived object: %w", err)
		}
		
		objectsInfo = append(objectsInfo, inf)
	}
	
	// Проверяем на ошибки, которые могли возникнуть во время итерации
	if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("PostgresStorageAdapter: error during archived rows iteration: %w", err)
    }

    return objectsInfo, nil 
}

func (a *PostgresStorageAdapter) GetObjectByIDForActualization(ctx context.Context, id string) (*domain.PropertyBasicInfo, error) {
    query := `SELECT id, source_ad_id, source, updated_at FROM general_properties 
               WHERE id = $1`
    
	rows, err := a.pool.Query(ctx, query, id)

	if err != nil {
		return nil, fmt.Errorf("PostgresStorageAdapter: failed to query object with id %s: %w", id, err)
	}
	defer rows.Close()


	var objectInfo domain.PropertyBasicInfo
	if rows.Next() {				
		if err := rows.Scan(&objectInfo.ID, &objectInfo.SourceAdID, &objectInfo.Source, &objectInfo.UpdatedAt); err != nil {
			return nil, fmt.Errorf("PostgresStorageAdapter: failed to scan object with id %s: %w", id, err)
		}		
	}
	
	 // Проверяем на ошибки, которые могли возникнуть во время итерации
	if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("PostgresStorageAdapter: error during rows iteration for object id = %s: %w", id, err)
    }

    return &objectInfo, nil
}