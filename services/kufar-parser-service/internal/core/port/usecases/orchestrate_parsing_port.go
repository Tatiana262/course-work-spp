package usecases_port

import (
	"context"
	"kufar-parser-service/internal/core/domain"

	"github.com/google/uuid"
)

type OrchestrateParsingPort interface {
	Execute(ctx context.Context, internalTasks []domain.SearchCriteria, taskID uuid.UUID) error
}