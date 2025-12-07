package usecases_port

import (
	"context"
	"storage-service/internal/core/domain"
)

type GetBestObjectsByMasterIDsUseCase interface {
	Execute(ctx context.Context, masterIDs []string) ([]domain.GeneralPropertyInfo, error)
}