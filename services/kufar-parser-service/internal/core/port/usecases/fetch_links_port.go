package usecases_port

import (
	"context"
	"kufar-parser-service/internal/core/domain"

	"github.com/google/uuid"
)

type FetchLinksPort interface {
	Execute(ctx context.Context, initialCriteria domain.SearchCriteria, taskID uuid.UUID) (int, error)
}