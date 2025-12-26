package usecase

import (
	"context"
	"fmt"

	// "kufar-parser-service/internal/constants"
	"kufar-parser-service/internal/contextkeys"
	"kufar-parser-service/internal/core/domain"
	"kufar-parser-service/internal/core/port"
	// "log"
	"time"

	"github.com/google/uuid"
)

const kufarParserName = "kufar_link_fetcher"

// FetchAndEnqueueLinksUseCase инкапсулирует логику получения ссылок и отправки их в очередь
type FetchAndEnqueueLinksUseCase struct {
	fetcherRepo  port.KufarFetcherPort
	queueRepo    port.LinksQueuePort
	lastRunRepo  port.LastRunRepositoryPort
	sourceName   string // Имя источника, например, "kufar"
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

// Execute запускает процесс получения и постановки ссылок в очередь.
// `initialCriteria` содержит базовые фильтры для первого запроса.
func (uc *FetchAndEnqueueLinksUseCase) Execute(ctx context.Context, initialCriteria domain.SearchCriteria, taskID uuid.UUID) (int, error) {
	
	baseLogger := contextkeys.LoggerFromContext(ctx)
	ucLogger := baseLogger.WithFields(port.Fields{
		"use_case": "FetchAndEnqueueLinks",
		// "task_id":  taskID,
		"source":   uc.sourceName,
	})

	ucLogger.Info("Starting to fetch links", port.Fields{"criteria": initialCriteria.Name})

	// Формируем уникальный ключ для parserName на основе критериев, чтобы хранить lastRunTime для каждой комбинации (на основе url?)
	parserNameKey := fmt.Sprintf("%s_%s_%s_%s",
		kufarParserName,
		initialCriteria.Category,
		initialCriteria.DealType,
		initialCriteria.Location,
	)

	parserLogger := ucLogger.WithFields(port.Fields{"parser_key": parserNameKey})

	lastRunTime, err := uc.lastRunRepo.GetLastRunTimestamp(ctx, parserNameKey) // Делаем ключ уникальным для комбинации фильтров
	if err != nil {
		// Если ошибка (например, нет записи), можем начать с "начала времен" или за определенный период назад
		parserLogger.Warn("Could not get last run timestamp, fetching from the beginning.", port.Fields{"error": err.Error()})
		lastRunTime = time.Time{} // Или, например, time.Now().Add(-24 * time.Hour)
	} else {
		parserLogger.Info("Last run timestamp found", port.Fields{"last_run_time": lastRunTime})
	}

	//FOR DEBUG
	// lastRunTime = time.Time{}
	const debugLinkLimit = 30 // Собираем не больше 5 ссылок для каждого вызова
	stopFetching := false

	currentCriteria := initialCriteria
	newLinksFoundOverall := 0
	totalPagesProcessed := 0
	var latestAdTimeOnCurrentRun time.Time // Для сохранения самой новой даты объявления в текущем запуске

	
	for {
		select {
        case <-ctx.Done():
            return 0, ctx.Err() // Прерываемся, если пришел сигнал о завершении
        default:
        }

			
		totalPagesProcessed++
		pageLogger := parserLogger.WithFields(port.Fields{
			"page":   totalPagesProcessed,
			"cursor": currentCriteria.Cursor,
		})
		pageLogger.Debug("Fetching page", nil)

		links, nextCursor, fetchErr := uc.fetcherRepo.FetchLinks(ctx, currentCriteria, lastRunTime)
		if fetchErr != nil {
			pageLogger.Error("Error fetching links from repository", fetchErr, nil)
			return 0, fmt.Errorf("use case: error fetching links for source '%s' with criteria %s: %w", uc.sourceName, currentCriteria.Name, fetchErr)
		}

		if len(links) == 0 && nextCursor == "" {
			// Это может случиться, если на первой же странице нет новых ссылок И нет следующей страницы,
			// или если адаптер сразу вернул пустой nextCursor из-за отсечки по 'since'
			pageLogger.Debug("No new links found and no next cursor. Stopping.", nil)
			break
		}
		
		newLinksOnPage := 0
		for _, link := range links {
			link.Source = uc.sourceName // Добавляем источник
			err = uc.queueRepo.Enqueue(ctx, link, taskID)
			if err != nil {
				// Здесь можно решить, что делать: пропустить ссылку, повторить, остановить всё
				pageLogger.Error("Error enqueuing link, skipping", err, port.Fields{"ad_id": link.AdID})
				continue // Пропускаем эту ссылку, но продолжаем с остальными
			}
			newLinksOnPage++
			newLinksFoundOverall++
			if link.ListedAt.After(latestAdTimeOnCurrentRun) { // Обновляем самое свежее время
				latestAdTimeOnCurrentRun = link.ListedAt
			}

			//FOR DEBUG
			if newLinksFoundOverall >= debugLinkLimit {
				parserLogger.Warn("DEBUG: Link limit reached. Stopping fetch process.", port.Fields{
					"limit": debugLinkLimit,
					"total_found": newLinksFoundOverall,
				})
				stopFetching = true // Устанавливаем флаг для выхода из внешнего цикла
				break // Выходим из внутреннего цикла `for _, link := ...`
			}
		}

		//FOR DEBUG
		if stopFetching {
			break 
		}

		if newLinksOnPage > 0 {
			pageLogger.Debug("Enqueued new links from page", port.Fields{"count": newLinksOnPage})
		}

		if nextCursor == "" {
			parserLogger.Debug("No next cursor. Pagination finished.", nil)
			break
		}

		// log.Printf("Use Case: Fetched %d new links from page. Next cursor: %s\n", newLinksOnPage, nextCursor)
		currentCriteria.Cursor = nextCursor

		// Опционально: добавить задержку между запросами страниц пагинации здесь, если
		// fetcherRepo не управляет этим сам (хотя Colly управляет).
		// time.Sleep(1 * time.Second)
	}

	// Обновляем lastRunTime, если были найдены новые ссылки
	if newLinksFoundOverall > 0 && !latestAdTimeOnCurrentRun.IsZero() {
		// Устанавливаем время самого нового объявления, которое мы обработали в этом запуске
		err = uc.lastRunRepo.SetLastRunTimestamp(ctx, parserNameKey, latestAdTimeOnCurrentRun)
		if err != nil {
			parserLogger.Error("Error setting last run timestamp", err, port.Fields{"new_timestamp": latestAdTimeOnCurrentRun})
		} else {
			parserLogger.Info("Successfully set last run timestamp", port.Fields{"new_timestamp": latestAdTimeOnCurrentRun})
		}
	} else if newLinksFoundOverall == 0 && totalPagesProcessed > 0 && !lastRunTime.IsZero() {
        // Если мы прошлись по страницам, но не нашли НИ ОДНОЙ НОВОЙ ссылки (т.е. все были отсеяны по 'since' в адаптере,
        // или их просто не было), но при этом lastRunTime не нулевой, то можно его обновить на текущее время,
        // чтобы показать, что мы проверяли. Но это опционально, можно и не обновлять, если ничего нового не было.
        currentTime := time.Now().UTC()
        uc.lastRunRepo.SetLastRunTimestamp(ctx, parserNameKey, currentTime)
		parserLogger.Info("No new links found, but checked. Updated last run to current time.", port.Fields{"new_timestamp": currentTime})
    }

	ucLogger.Info("Finished fetching links", port.Fields{
		"total_links_enqueued": newLinksFoundOverall,
		"total_pages_processed": totalPagesProcessed,
	})

	return newLinksFoundOverall, nil
}

