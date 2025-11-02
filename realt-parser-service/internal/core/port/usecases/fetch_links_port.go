package usecases_port

import (
	"context"
	"realt-parser-service/internal/core/domain"
)

type FetchLinksPort interface {
	Execute(ctx context.Context, initialCriteria domain.SearchCriteria) error
}