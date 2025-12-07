package port

import (
	"context"
	"favorites-service/internal/core/domain" // Предполагаем, что здесь будет структура ObjectCard
	"github.com/google/uuid"
)

// ObjectStoragePort - контракт для клиента, который общается с storage-service.
type ObjectStoragePort interface {
	// Получает обогащенные данные по списку master_object_id.
	GetBestObjectsByMasterIDs(ctx context.Context, masterIDs []uuid.UUID) ([]domain.ObjectCard, error)
}