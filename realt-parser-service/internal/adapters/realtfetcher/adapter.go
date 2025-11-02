package realtfetcher

import (
	"log"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
)

// RealtFetcherAdapter отвечает за все взаимодействия с сайтом Realt
type RealtFetcherAdapter struct {
	collector *colly.Collector
	baseURL   string
}

// NewRealtFetcherAdapter - конструктор
func NewRealtFetcherAdapter(baseURL string) *RealtFetcherAdapter {

	// родительский коллектор
	c := colly.NewCollector(colly.AllowedDomains("realt.by"), colly.AllowURLRevisit())

	err := c.Limit(&colly.LimitRule{
		DomainGlob: "realt.by",
		Parallelism: 1,
		RandomDelay: 3 * time.Second,
	})
	if err != nil {
		log.Fatalf("RealtFetcherAdapter: Failed to set limit rule: %v", err)
	}

	
	extensions.RandomUserAgent(c) // На каждый запрос будет подставлен User-Agent реального браузера
	extensions.Referer(c)         // Автоматически подставляет заголовок Referer, имитируя навигацию


	c.OnError(func(r *colly.Response, err error) {
		log.Printf("RealtFetcherAdapter: Error during request to %s: Status=%d, Error=%v", r.Request.URL, r.StatusCode, err)
	})
	c.OnRequest(func(r *colly.Request) {
		log.Printf("RealtFetcherAdapter: Making request to %s", r.URL.String())
	})

	return &RealtFetcherAdapter{
		collector: c,
		baseURL:   baseURL,
	}
}