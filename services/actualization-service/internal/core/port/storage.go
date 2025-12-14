package port

import (
	"actualization-service/internal/core/domain"
	"context"
)

type StoragePort interface {
	GetActiveObjects(ctx context.Context, category string, limit int) ([]domain.PropertyInfo, error)
	GetArchivedObjects(ctx context.Context, category string, limit int) ([]domain.PropertyInfo, error)
	GetObjectsByMasterID(ctx context.Context, master_id string) ([]domain.PropertyInfo, error)
}
