package usecase

import (
	"context"
	"fmt"
	"log"
	"kufar-parser-service/internal/core/domain"
	"kufar-parser-service/internal/core/port"
	"time"
)

const kufarParserName = "kufar_link_fetcher"

// FetchAndEnqueueLinksUseCase инкапсулирует логику получения ссылок и отправки их в очередь
type FetchAndEnqueueLinksUseCase struct {
	fetcherRepo  port.KufarFetcherPort
	queueRepo    port.LinksQueuePort
	lastRunRepo  port.LastRunRepositoryPort
	sourceName   string
}

// NewFetchAndEnqueueLinksUseCase создает новый экземпляр FetchAndEnqueueLinksUseCase
func NewFetchAndEnqueueLinksUseCase(
	fetcher port.KufarFetcherPort,
	queue port.LinksQueuePort,
	lastRun port.LastRunRepositoryPort,
	sourceName string,
) *FetchAndEnqueueLinksUseCase {
	return &FetchAndEnqueueLinksUseCase{
		fetcherRepo: fetcher,
		queueRepo:   queue,
		lastRunRepo: lastRun,
		sourceName:  sourceName,
	}
}

// Execute запускает процесс получения и постановки ссылок в очередь
func (uc *FetchAndEnqueueLinksUseCase) Execute(ctx context.Context, initialCriteria domain.SearchCriteria) error {
	log.Printf("Use Case: Starting to fetch links for source '%s' with initial criteria: %+v\n", uc.sourceName, initialCriteria)

	parserNameKey := fmt.Sprintf("%s_%s_%s_%s",
		kufarParserName,
		initialCriteria.Category,
		initialCriteria.DealType,
		initialCriteria.Location,
	)

	lastRunTime, err := uc.lastRunRepo.GetLastRunTimestamp(ctx, parserNameKey)
	if err != nil {
		log.Printf("Use Case: Could not get last run timestamp for '%s': %v. Fetching from a default point in time (or very old).\n", parserNameKey, err)
		lastRunTime = time.Time{}
	} else {
		log.Printf("Use Case: Last run for '%s' was at %s. Fetching links since then.\n", parserNameKey, lastRunTime.Format(time.RFC3339))
	}

	//FOR DEBUG
	//lastRunTime = time.Time{}

	currentCriteria := initialCriteria
	newLinksFoundOverall := 0
	totalPagesProcessed := 0
	var latestAdTimeOnCurrentRun time.Time // Для сохранения самой новой даты объявления в текущем запуске

	for {
		select {
        case <-ctx.Done():
            return ctx.Err() 
        default:
        }
		
		totalPagesProcessed++
		log.Printf("Use Case: Fetching page %d with cursor '%s'\n", totalPagesProcessed, currentCriteria.Cursor)

		links, nextCursor, fetchErr := uc.fetcherRepo.FetchLinks(ctx, currentCriteria, lastRunTime)
		if fetchErr != nil {
			return fmt.Errorf("use case: error fetching links for source '%s' with criteria %+v: %w", uc.sourceName, currentCriteria, fetchErr)
		}

		if len(links) == 0 && nextCursor == "" {
			log.Printf("Use Case: No links found and no next cursor for source '%s'. Stopping.\n", uc.sourceName)
			break
		}
		
		newLinksOnPage := 0
		for _, link := range links {
			link.Source = uc.sourceName
			err = uc.queueRepo.Enqueue(ctx, link)
			if err != nil {
				log.Printf("Use Case: Error enqueuing link with AdID %d for source '%s': %v. Skipping this link.\n", link.AdID, uc.sourceName, err)
				continue // Пропускаем эту ссылку, но продолжаем с остальными
			}
			newLinksOnPage++
			newLinksFoundOverall++
			if link.ListedAt.After(latestAdTimeOnCurrentRun) { 
				latestAdTimeOnCurrentRun = link.ListedAt
			}
			log.Printf("Use Case: Enqueued link with AdID: %d (ListedAt: %s)\n", link.AdID, link.ListedAt.Format(time.RFC3339))
		}

		if nextCursor == "" {
			log.Printf("Use Case: No next cursor for source '%s'. Pagination finished for this criteria set.\n", uc.sourceName)
			break
		}

		log.Printf("Use Case: Fetched %d new links from page. Next cursor: %s\n", newLinksOnPage, nextCursor)
		currentCriteria.Cursor = nextCursor
	}

	// Обновляем lastRunTime, если были найдены новые ссылки
	if newLinksFoundOverall > 0 && !latestAdTimeOnCurrentRun.IsZero() {
		// Устанавливаем время самого нового объявления, которое мы обработали в этом запуске
		err = uc.lastRunRepo.SetLastRunTimestamp(ctx, parserNameKey, latestAdTimeOnCurrentRun)
		if err != nil {
			log.Printf("Use Case: Error setting last run timestamp for '%s' (key: %s) to %s: %v\n", uc.sourceName, parserNameKey, latestAdTimeOnCurrentRun.Format(time.RFC3339), err)
		} else {
			log.Printf("Use Case: Successfully set last run timestamp for '%s' (key: %s) to %s\n", uc.sourceName, parserNameKey, latestAdTimeOnCurrentRun.Format(time.RFC3339))
		}
	} else if newLinksFoundOverall == 0 && totalPagesProcessed > 0 && !lastRunTime.IsZero() {
        currentTime := time.Now().UTC()
        uc.lastRunRepo.SetLastRunTimestamp(ctx, parserNameKey, currentTime)
        log.Printf("Use Case: No new links found for '%s' (key: %s), but checked. Updated last run to %s.\n", uc.sourceName, parserNameKey, currentTime.Format(time.RFC3339))
    }

	log.Printf("Use Case: Finished fetching links for source '%s'. Total new links enqueued: %d. Total pages processed: %d\n", uc.sourceName, newLinksFoundOverall, totalPagesProcessed)
	return nil
}