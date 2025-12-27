package usecase

import (
	"context"
	"fmt"
	"kufar-parser-service/internal/contextkeys"
	"kufar-parser-service/internal/core/domain"
	"kufar-parser-service/internal/core/port"

	"github.com/google/uuid"
)

// ProcessLinkUseCase инкапсулирует логику обработки одной ссылки:
// парсинг деталей и отправка результата в следующую очередь
type ProcessLinkUseCase struct {
	detailsFetcher port.KufarFetcherPort
	resultQueue    port.ProcessedPropertyQueuePort
}

// NewProcessLinkUseCase создает новый экземпляр use case
func NewProcessLinkUseCase(
	fetcher port.KufarFetcherPort,
	queue port.ProcessedPropertyQueuePort,
) *ProcessLinkUseCase {
	return &ProcessLinkUseCase{
		detailsFetcher: fetcher,
		resultQueue:    queue,
	}
}

// Execute выполняет основную логику use case
func (uc *ProcessLinkUseCase) Execute(ctx context.Context, linkToParse domain.PropertyLink, taskID uuid.UUID) error {

	baseLogger := contextkeys.LoggerFromContext(ctx)
	ucLogger := baseLogger.WithFields(port.Fields{
		"use_case": "ProcessLink",
		// "ad_id":    linkToParse.AdID,
		// "task_id":  taskID,
	})
	
	ucLogger.Debug("Processing link", nil)

	// Используем порт для парсинга деталей
	propertyRecord, fetchErr := uc.detailsFetcher.FetchAdDetails(ctx, linkToParse.AdID)
	
	if fetchErr != nil {
		ucLogger.Error("Failed to fetch/parse details", fetchErr, nil)
		return fmt.Errorf("failed to fetch/parse details for %d: %w", linkToParse.AdID, fetchErr)
	}

	if propertyRecord.General.Status == domain.StatusArchived {
		ucLogger.Debug("Successfully processed as archived.", nil)
	} else {
		ucLogger.Debug("Successfully parsed details.", nil)
	}


	// Используем порт для отправки результата в очередь
	err := uc.resultQueue.Enqueue(ctx, *propertyRecord, taskID)
	if err != nil {
		ucLogger.Error("failed to enqueue processed data", err, nil)
		return fmt.Errorf("CRITICAL: failed to enqueue processed data for AdID %d: %w", linkToParse.AdID, err)
	}

	// log.Printf("ProcessLinkUseCase: Successfully enqueued processed data for '%d'.\n", linkToParse.AdID)
	ucLogger.Info("Successfully enqueued processed data", nil)
	return nil
}