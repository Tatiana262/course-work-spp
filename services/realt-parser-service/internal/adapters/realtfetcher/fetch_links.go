package realtfetcher

import (
	"context"
	"encoding/json"
	"fmt"
	// "log"
	"realt-parser-service/internal/constants"
	"realt-parser-service/internal/contextkeys"
	"realt-parser-service/internal/core/domain"
	"realt-parser-service/internal/core/port"
	"time"

	"github.com/gocolly/colly/v2"
)

// Структуры для парсинга ответа от Realt.by
type responseRoot struct { Data struct { SearchObjectsV2 struct { Body listingBody `json:"body"` } `json:"searchObjectsV2"` } `json:"data"` }
type listingBody struct { Results []adItem `json:"results"`; Pagination pagination `json:"pagination"` }
type adItem struct { Code int64 `json:"code"`; UpdatedAt time.Time `json:"updatedAt"` }
type pagination struct { Page int `json:"page"`; PageSize int `json:"pageSize"`; TotalCount int `json:"totalCount"` }


type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables RequestVariables		 `json:"variables"`
}

const query = `
	query searchObjectsV2($data: GetObjectsByAddressInputV2!) {
		searchObjectsV2(data: $data) {
		  ...StatusAndErrors
		  body {
			results {
			  code
			  updatedAt
			}
		   pagination {
			  page
			  pageSize
			  totalCount
			}
			  
		  }
		}
	  }
	  
	  fragment StatusAndErrors on INullResponse {
		success
		errors {
		  code
		  title
		  message
		  field
		}
	  }
	`

// FetchLinks возвращает ссылки, номер следующей страницы (или 0, если страниц больше нет) и ошибку.
func (a *RealtFetcherAdapter) FetchLinks(ctx context.Context, criteria domain.SearchCriteria, since time.Time) ([]domain.PropertyLink, int, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	fetchLinksLogger := logger.WithFields(port.Fields{"component": "RealtFetcherAdapter(FetchLinks)"})
	
	collector := a.collector.Clone()
	var fetchedLinks []domain.PropertyLink
	var responseErr error
	var nextPageNum int
	var stopProcessing bool = false

	variables := buildGraphQLVariables(criteria)
	
	 // Создаем тело запроса
	 requestBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil { return nil, 0, fmt.Errorf("realt adapter: failed to marshal variables: %w", err) }

	collector.OnRequest(func(r *colly.Request) { 
		fetchLinksLogger.Debug("Making request to fetch links", port.Fields{
			"url": r.URL.String(),
		})

		r.Headers.Set("Content-Type", "application/json") 
	})

	collector.OnResponse(func(r *colly.Response) {
		var data responseRoot
		if err := json.Unmarshal(r.Body, &data); err != nil {
			fetchLinksLogger.Error("failed to unmarshal links json", err, port.Fields{
				"criteria": criteria.Name,
			})
			responseErr = fmt.Errorf("realt adapter: failed to unmarshal json: %w", err)
			return
		}

		p := data.Data.SearchObjectsV2.Body.Pagination
		for _, ad := range data.Data.SearchObjectsV2.Body.Results {

			// Если объявление старше или равно 'since', устанавливаем флаг остановки и прекращаем цикл.
			if !since.IsZero() && (ad.UpdatedAt.Before(since) || ad.UpdatedAt.Equal(since)) {
				fetchLinksLogger.Debug("Reached the 'since' date cutoff", port.Fields{ // <-- Используем logger
					"since_date": since.Format(time.RFC3339),
					"ad_code":    ad.Code,
				})
				stopProcessing = true
				break 
			}

			// var url string
			// if (criteria.DealType == "") {
			// 	url = fmt.Sprintf("https://realt.by/%s/object/%d/", criteria.PropertyType, ad.Code)
			// } else {
			// 	url = fmt.Sprintf("https://realt.by/%s-%s/object/%d/", criteria.DealType, criteria.PropertyType, ad.Code)
			// }

			fetchedLinks = append(fetchedLinks, domain.PropertyLink{
				AdID:   ad.Code,
				Source:   "realt.by",
				ListedAt: ad.UpdatedAt,
				URL:      fmt.Sprintf("https://realt.by/%s/object/%d/", constants.SearchConfigs[criteria.Category], ad.Code),
			})
		}
		
		if !stopProcessing && p.Page*p.PageSize < p.TotalCount {
			nextPageNum = p.Page + 1
		}
	})
	
	collector.OnError(func(r *colly.Response, err error) {
		fetchLinksLogger.Error("Failed to fetch links page", err, port.Fields{
			"url":    r.Request.URL.String(),
			"status": r.StatusCode,
		})
		responseErr = fmt.Errorf("realt adapter: request to %s failed with status %d: %w", r.Request.URL, r.StatusCode, err)
	})

	if err := collector.PostRaw(a.baseURL, jsonData); err != nil {
		fetchLinksLogger.Error("Failed to post request", err, port.Fields{"url": a.baseURL})
		return nil, 0, fmt.Errorf("realt adapter: failed to post request: %w", err)
	}
	collector.Wait()

	if responseErr != nil { return nil, 0, responseErr }

	if stopProcessing {
		fetchLinksLogger.Info("Link processing stopped due to 'since' date.", nil)
		nextPageNum = 0 // Не переходим дальше, если остановились
	}
	
	fetchLinksLogger.Info("Finished fetching links for page", port.Fields{
		"page":            criteria.Page,
		"links_fetched": len(fetchedLinks),
		"next_page":   nextPageNum,
	})
	
	return fetchedLinks, nextPageNum, nil
}