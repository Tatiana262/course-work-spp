package usecases_port

import (
	"context"
	"kufar-parser-service/internal/core/domain"

	"github.com/google/uuid"
)

type ProcessLinkPort interface {
	Execute(ctx context.Context, linkToParse domain.PropertyLink, taskID uuid.UUID) error
}