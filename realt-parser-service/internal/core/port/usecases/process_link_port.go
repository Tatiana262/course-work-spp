package usecases_port

import (
	"context"
	"realt-parser-service/internal/core/domain"
)

type ProcessLinkPort interface {
	Execute(ctx context.Context, linkToParse domain.PropertyLink) error
}