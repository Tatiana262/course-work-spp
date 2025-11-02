package usecase

import (
	"context"
	"fmt"
	"log"
	"realt-parser-service/internal/core/domain"
	"realt-parser-service/internal/core/port"
)

// ProcessLinkUseCase инкапсулирует логику обработки одной ссылки:
// парсинг деталей и отправка результата в следующую очередь
type ProcessLinkUseCase struct {
	detailsFetcher port.RealtFetcherPort
	resultQueue    port.ProcessedPropertyQueuePort
}

// NewProcessLinkUseCase создает новый экземпляр use case.
func NewProcessLinkUseCase(
	fetcher port.RealtFetcherPort,
	queue port.ProcessedPropertyQueuePort,
) *ProcessLinkUseCase {
	return &ProcessLinkUseCase{
		detailsFetcher: fetcher,
		resultQueue:    queue,
	}
}


func (uc *ProcessLinkUseCase) Execute(ctx context.Context, linkToParse domain.PropertyLink) error {

	log.Printf("ProcessLinkUseCase: Processing AdID: %d\n", linkToParse.AdID)

	// парсинг деталей
	propertyRecord, fetchErr := uc.detailsFetcher.FetchAdDetails(ctx, linkToParse.URL)
	if fetchErr != nil {
		return fmt.Errorf("failed to fetch/parse details for %d: %w", linkToParse.AdID, fetchErr)
	}

	log.Printf("ProcessLinkUseCase: Successfully parsed details for AdID %d. Title: %s\n", linkToParse.AdID, propertyRecord.General.Title)

	// отправка результата в очередь
	err := uc.resultQueue.Enqueue(ctx, *propertyRecord)
	if err != nil {
		return fmt.Errorf("CRITICAL: failed to enqueue processed data for AdID %d: %w", linkToParse.AdID, err)
	}

	log.Printf("ProcessLinkUseCase: Successfully enqueued processed data for '%d'.\n", linkToParse.AdID)
	return nil
}