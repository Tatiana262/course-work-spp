package kufarfetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"kufar-parser-service/internal/contextkeys"
	"kufar-parser-service/internal/core/domain"
	"kufar-parser-service/internal/core/port"

	// "log"
	"net/url"
	"strconv"

	// "parser-project/internal/core/port" // Не нужен здесь, т.к. интерфейс в другом месте
	// "strings"
	"time"

	"github.com/gocolly/colly/v2"
)

// ... (kufarRoot, kufarProps и т.д. остаются такими же) ...
// type kufarRoot struct {
// 	Props kufarProps `json:"props"`
// }

// type kufarProps struct {
// 	InitialState kufarInitialState `json:"initialState"`
// }

// type kufarInitialState struct {
// 	Listings kufarListings `json:"listing"`
// }

type kufarListings struct {
	Ads        []kufarAdItem        `json:"ads"`
	Pagination kufarPages 			`json:"pagination"`
}

type kufarAdItem struct {
	AdId     int64 `json:"ad_id"`
	AdLink   string `json:"ad_link"`
	ListTime string `json:"list_time"`
}

type kufarPages struct {
	Pages []kufarPaginationItem 	`json:"pages"`
}

type kufarPaginationItem struct {
	Label string      `json:"label"`
	Num   json.Number `json:"num"` 
	Token *string     `json:"token"`
}


func (a *KufarFetcherAdapter) buildURLFromCriteria(criteria domain.SearchCriteria) (string, error) {

	u, err := url.Parse(a.baseURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	if criteria.Category != "" {
		q.Set("cat", criteria.Category)
	}
	if criteria.DealType != "" {
		q.Set("typ", criteria.DealType)
	}
	if criteria.Location != "" {
		q.Set("gtsy", criteria.Location)
	}
	if criteria.AdsAmount != 0 {
		q.Set("size", strconv.Itoa(criteria.AdsAmount))
	}
	if criteria.SortBy != "" {
		q.Set("sort", criteria.SortBy)
	}
	if criteria.Cursor != "" {
		q.Set("cursor", criteria.Cursor)
	}
	if criteria.Query != "" {
		q.Set("bkbt", criteria.Query)
	}

	// FOR DEBUG
	//можно добавить query параметры для тестирования, например rms=v.or%3A5

	// для квартир
	// q.Set("rms", "v.or:1")

	// для домов
	// q.Set("rms", "v.or:5")
	// q.Set("prc", "r:0,20000000")
	
	// для коммерческой
	// q.Set("prc", "r:1000,2000")

	u.RawQuery = q.Encode()
	return u.String(), nil

}

func (a *KufarFetcherAdapter) FetchLinks(ctx context.Context, criteria domain.SearchCriteria, since time.Time) ([]domain.PropertyLink, string, error) {
	// Создаем "одноразовый" клон для этого конкретного запроса
	// Он наследует лимиты, но имеет свои собственные обработчики!
	logger := contextkeys.LoggerFromContext(ctx)
	fetchLinksLogger := logger.WithFields(port.Fields{"component": "KufarFetcherAdapter(FetchLinks)"})

	collector := a.collector.Clone()

	var fetchedLinks []domain.PropertyLink
	var nextCursor string
	var responseErr error // Для хранения ошибки из колбэка
	var stopProcessing bool = false
	
	targetURL, err := a.buildURLFromCriteria(criteria)
	if err != nil {
		return nil, "", fmt.Errorf("kufar adapter: failed to build URL from criteria: %w", err)
	}

	collector.OnRequest(func(r *colly.Request) {
		fetchLinksLogger.Debug("Making request to fetch links", port.Fields{
			"url": r.URL.String(),
		})
	})

	collector.OnResponse(func(r *colly.Response) {
		
		// Десериализуем JSON из тела ответа
		var data kufarListings
		jsonErr := json.Unmarshal(r.Body, &data)
		if jsonErr != nil {
			responseErr = fmt.Errorf("KufarAdapter: Ошибка при разборе JSON на странице %s: %w", r.Request.URL.String(), jsonErr)
			return
		}

		// var pageLinks []domain.PropertyLink // Ссылки, собранные с этой страницы
		// localStop := false

		if data.Ads != nil {
			for _, ad := range data.Ads {
				listedAt, parseErr := time.Parse(time.RFC3339, ad.ListTime)
				if parseErr != nil {
					fetchLinksLogger.Warn("Failed to parse date, skipping ad", port.Fields{ // <-- Используем logger
						"date_string": ad.ListTime,
						"ad_link":     ad.AdLink,
						"error":       parseErr.Error(),
					})
					continue
				}

				// Если объявление старше или равно 'since', устанавливаем флаг остановки и прекращаем цикл.
				if !since.IsZero() && (listedAt.Before(since) || listedAt.Equal(since)) {
					fetchLinksLogger.Debug("Reached the 'since' date cutoff", port.Fields{ // <-- Используем logger
						"since_date": since.Format(time.RFC3339),
						"ad_link":    ad.AdLink,
					})
					stopProcessing = true
					break 
				}
				fetchedLinks = append(fetchedLinks, domain.PropertyLink{ListedAt: listedAt, AdID: ad.AdId})
			}
		}
		
		// Ищем токен пагинации, только если не нужно останавливаться
		if !stopProcessing && data.Pagination.Pages != nil {
			for _, pItem := range data.Pagination.Pages {
				if pItem.Label == "next" && pItem.Token != nil && *pItem.Token != "" {
					nextCursor = *pItem.Token
					break
				}
			}
		}
	})

	collector.OnError(func(r *colly.Response, err error) {
		fetchLinksLogger.Error("Failed to fetch links page", err, port.Fields{
			"url":    r.Request.URL.String(),
			"status": r.StatusCode,
		})
		responseErr = fmt.Errorf("KufarAdapter: request to %s failed with status %d: %w", r.Request.URL, r.StatusCode, err)
	})
	
	// Ошибки обрабатываются в глобальном OnError, но мы все равно должны
	// проверить ошибку самого вызова Visit (например, если домен не разрешен)
	visitErr := collector.Visit(targetURL)
	if visitErr != nil {
		fetchLinksLogger.Error("Failed to initiate visit for fetching links", visitErr, port.Fields{"url": targetURL})
		return nil, "", fmt.Errorf("kufar adapter: failed to visit URL %s: %w", targetURL, visitErr)
	}
	collector.Wait()

	// Если внутри колбэка произошла ошибка, возвращаем ее
	if responseErr != nil {
		return nil, "", responseErr
	}

	if stopProcessing {
		fetchLinksLogger.Info("Link processing stopped due to 'since' date.", port.Fields{
			"since": since,
		})
		nextCursor = "" // Не переходим дальше, если остановились
	}
	
	fetchLinksLogger.Info("Finished fetching links for URL", port.Fields{
		"url":           targetURL,
		"links_fetched": len(fetchedLinks),
		"next_cursor":   nextCursor,
	})
	
	return fetchedLinks, nextCursor, nil
}