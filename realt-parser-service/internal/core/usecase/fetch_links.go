package usecase

import (
	"context"
	"fmt"
	"log"
	"realt-parser-service/internal/core/domain"
	"realt-parser-service/internal/core/port"
	"time"
)

const realtParserName = "realt_link_fetcher"

// FetchAndEnqueueLinksUseCase инкапсулирует логику получения ссылок и отправки их в очередь
type FetchAndEnqueueLinksUseCase struct {
	fetcherRepo  port.RealtFetcherPort
	queueRepo    port.LinksQueuePort
	lastRunRepo  port.LastRunRepositoryPort
	sourceName   string 
}

// NewFetchAndEnqueueLinksUseCase создает новый экземпляр FetchAndEnqueueLinksUseCase
func NewFetchAndEnqueueLinksUseCase(
	fetcher port.RealtFetcherPort,
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

	// уникальный ключ для на основе критериев, чтобы хранить lastRunTime для каждой комбинации
	parserNameKey := fmt.Sprintf("%s_%d_%s",
		realtParserName,
		initialCriteria.Category,
		initialCriteria.LocationUUID,
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
		log.Printf("Use Case: Fetching page %d\n", currentCriteria.Page)

		links, nextPage, fetchErr := uc.fetcherRepo.FetchLinks(ctx, currentCriteria, lastRunTime)
		if fetchErr != nil {
			return fmt.Errorf("use case: error fetching links for source '%s' with criteria %+v: %w", uc.sourceName, currentCriteria, fetchErr)
		}

		if len(links) == 0 && nextPage == 0 {
			log.Printf("Use Case: No links found and no next page for source '%s'. Stopping.\n", uc.sourceName)
			break
		}
		
		newLinksOnPage := 0
		for _, link := range links {
			link.Source = uc.sourceName 
			err = uc.queueRepo.Enqueue(ctx, link)
			if err != nil {
				log.Printf("Use Case: Error enqueuing link with AdID %d for source '%s': %v. Skipping this link.\n", link.AdID, uc.sourceName, err)
				continue 
			}
			newLinksOnPage++
			newLinksFoundOverall++
			if link.ListedAt.After(latestAdTimeOnCurrentRun) { 
				latestAdTimeOnCurrentRun = link.ListedAt
			}
			log.Printf("Use Case: Enqueued link with AdID: %d (ListedAt: %s)\n", link.AdID, link.ListedAt.Format(time.RFC3339))
		}

		if nextPage == 0 {
			log.Printf("Use Case: No next page for source '%s'. Pagination finished for this criteria set.\n", uc.sourceName)
			break
		}

		log.Printf("Use Case: Fetched %d new links from page. Next page: %d\n", newLinksOnPage, nextPage)
		currentCriteria.Page = nextPage
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