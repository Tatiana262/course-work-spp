package port

import (
	"context"
	"realt-parser-service/internal/core/domain"

	"github.com/google/uuid"
)

type TaskReporterPort interface {
	ReportResults(ctx context.Context, taskID uuid.UUID, stats *domain.ParsingTasksStats) error
}