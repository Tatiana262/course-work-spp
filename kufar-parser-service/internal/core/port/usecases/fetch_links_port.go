package usecases_port

import (
	"context"
	"kufar-parser-service/internal/core/domain"
)

type FetchLinksPort interface {
	Execute(ctx context.Context, initialCriteria domain.SearchCriteria) error
}