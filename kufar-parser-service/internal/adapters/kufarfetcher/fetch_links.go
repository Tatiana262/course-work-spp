package kufarfetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"kufar-parser-service/internal/core/domain"
	"strconv"

	"time"

	"github.com/gocolly/colly/v2"
)

type kufarListings struct {
	Ads        []kufarAdItem        `json:"ads"`
	Pagination kufarPages 			`json:"pagination"`
}

type kufarAdItem struct {
	AdId     int `json:"ad_id"`
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

	//можно добавить query параметры для тестирования, например rms=v.or%3A5
	q.Set("rms", "v.or:5")
	

	u.RawQuery = q.Encode()
	return u.String(), nil

}

func (a *KufarFetcherAdapter) FetchLinks(ctx context.Context, criteria domain.SearchCriteria, since time.Time) ([]domain.PropertyLink, string, error) {

	// наследует лимиты, но имеет свои собственные обработчики
	collector := a.collector.Clone()

	var fetchedLinks []domain.PropertyLink
	var nextCursor string
	var responseErr error 
	var stopProcessing bool = false
	
	targetURL, err := a.buildURLFromCriteria(criteria)
	if err != nil {
		return nil, "", fmt.Errorf("kufar adapter: failed to build URL from criteria: %w", err)
	}

	collector.OnResponse(func(r *colly.Response) {
	
		var data kufarListings
		jsonErr := json.Unmarshal(r.Body, &data)
		if jsonErr != nil {
			responseErr = fmt.Errorf("KufarAdapter: Ошибка при разборе JSON на странице %s: %w", r.Request.URL.String(), jsonErr)
			return
		}


		if data.Ads != nil {
			for _, ad := range data.Ads {
				listedAt, parseErr := time.Parse(time.RFC3339, ad.ListTime)
				if parseErr != nil {
					log.Printf("KufarAdapter: Ошибка парсинга даты '%s' для URL %s: %v. Пропускаем.\n", ad.ListTime, ad.AdLink, parseErr)
					continue
				}

				// Если объявление старше или равно since, устанавливаем флаг остановки и прекращаем цикл
				if !since.IsZero() && (listedAt.Before(since) || listedAt.Equal(since)) {
					log.Printf("KufarAdapter: Достигнута дата отсечки (%s) для URL %s.\n", since.Format(time.RFC3339), ad.AdLink)
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
		responseErr = fmt.Errorf("KufarAdapter: request to %s failed with status %d: %w", r.Request.URL, r.StatusCode, err)
	})
	
	visitErr := collector.Visit(targetURL)
	if visitErr != nil {
		return nil, "", fmt.Errorf("kufar adapter: failed to visit URL %s: %w", targetURL, visitErr)
	}
	collector.Wait()

	if responseErr != nil {
		return nil, "", responseErr
	}

	if stopProcessing {
		log.Println("KufarAdapter: Обработка остановлена из-за достижения 'since' или отмены контекста.")
		nextCursor = "" // Не переходим дальше, если остановились
	}
	
	log.Printf("KufarAdapter: Завершено для URL %s. Ссылок: %d. Следующий курсор: '%s'\n", targetURL, len(fetchedLinks), nextCursor)
	return fetchedLinks, nextCursor, nil
}