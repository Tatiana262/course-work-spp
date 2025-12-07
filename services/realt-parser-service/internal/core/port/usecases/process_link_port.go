package usecases_port

import (
	"context"
	"realt-parser-service/internal/core/domain"

	"github.com/google/uuid"
)

type ProcessLinkPort interface {
	Execute(ctx context.Context, link domain.PropertyLink, taskID uuid.UUID) error
}