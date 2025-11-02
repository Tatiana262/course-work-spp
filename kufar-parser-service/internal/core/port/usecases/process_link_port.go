package usecases_port

import (
	"context"
	"kufar-parser-service/internal/core/domain"
)

type ProcessLinkPort interface {
	Execute(ctx context.Context, linkToParse domain.PropertyLink) error
}