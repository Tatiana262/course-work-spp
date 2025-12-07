package usecases_port

import (
	"context"
	"storage-service/internal/core/domain"
)

type GetDictionariesUseCase interface {
	Execute(ctx context.Context, names []string) (map[string][]domain.DictionaryItem, error)
}