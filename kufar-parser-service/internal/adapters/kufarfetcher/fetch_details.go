package kufarfetcher

import (
	"context"
	"fmt"
	"log"
	// "os"
	"kufar-parser-service/internal/core/domain"
	// "time"

	"github.com/gocolly/colly/v2"
)


// FetchAdDetails извлекает и преобразует детальную информацию об объявлении
func (a *KufarFetcherAdapter) FetchAdDetails(ctx context.Context, adID int) (*domain.RealEstateRecord, error) {
	collector := a.collector.Clone()

	var record *domain.RealEstateRecord
	var fetchErr error

	collector.OnResponse(func(r *colly.Response) {

		rec, err := toDomainRecord(r.Body, "kufar")
		if err != nil {
			fetchErr = fmt.Errorf("FetchAdDetails: failed to map response to domain record: %w", err)
			return
		}
		record = rec
	})


	collector.OnError(func(r *colly.Response, err error) {
        log.Printf("FetchAdDetails failed for ad_id %d: %v", adID, err)
        fetchErr = err 
    })

	// Формируем URL для API, используя adID
	apiURL := fmt.Sprintf("https://api.kufar.by/search-api/v2/item/%d/rendered", adID)
	visitErr := collector.Visit(apiURL)
	if visitErr != nil {
		return nil, fmt.Errorf("kufar adapter (Detail): failed to visit URL %s: %w", apiURL, visitErr)
	}
	collector.Wait() 

	return record, fetchErr
}