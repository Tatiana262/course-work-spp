package realtfetcher

import (
	"context"
	"fmt"
	"log"
	// "encoding/json"
	// "net/url"
	// "os"
	// "path"
	"realt-parser-service/internal/core/domain"

	// "strings"

	"github.com/gocolly/colly/v2"
)


func (a *RealtFetcherAdapter) FetchAdDetails(ctx context.Context, adURL string) (*domain.RealEstateRecord, error) {
	collector := a.collector.Clone()

	var record *domain.RealEstateRecord
	var fetchErr error
	var rawJson string

	collector.OnHTML("script#__NEXT_DATA__", func(e *colly.HTMLElement) {	
		rawJson = e.Text
	})

	collector.OnError(func(r *colly.Response, err error) {
		log.Printf("FetchAdDetails failed for URL %s: %v", adURL, err)
        fetchErr = err 
	})

	// вызывается после того, как все OnHTML отработали
	collector.OnScraped(func(r *colly.Response) {
		if rawJson == "" {
			log.Fatal("Не удалось найти JSON данные на странице.")
			return
		}

		rec, err := toDomainRecord(rawJson, adURL, "realt")
		if err != nil {
			fetchErr = fmt.Errorf("FetchAdDetails: failed to map response to domain record: %w", err)
			return
		}
		record = rec
		
	})

	visitErr := collector.Visit(adURL)
	if visitErr != nil {
		return nil, fmt.Errorf("FetchAdDetails: failed to visit URL %s: %w", adURL, visitErr)
	}
	collector.Wait() 

	return record, fetchErr
}





