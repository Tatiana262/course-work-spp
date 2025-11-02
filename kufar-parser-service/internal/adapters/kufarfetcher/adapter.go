package kufarfetcher

import (
	"log"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
)

// KufarFetcherAdapter отвечает за все взаимодействия с kufar
type KufarFetcherAdapter struct {
	// один родительский коллектор, который разделяет лимиты
	collector *colly.Collector
	baseURL   string
}

// NewKufarFetcherAdapter - конструктор
func NewKufarFetcherAdapter(baseURL string) *KufarFetcherAdapter {
	
	c := colly.NewCollector(colly.AllowedDomains("api.kufar.by"), colly.AllowURLRevisit())

	err := c.Limit(&colly.LimitRule{	
		DomainGlob: "api.kufar.by",
		Parallelism: 1,
		RandomDelay: 3 * time.Second,
	})
	if err != nil {
		log.Fatalf("KufarFetcherAdapter: Failed to set limit rule: %v", err)
	}

	
	extensions.RandomUserAgent(c) // На каждый запрос будет подставлен User-Agent реального браузера
	extensions.Referer(c)         // Автоматически подставляет заголовок Referer, имитируя навигацию

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("KufarFetcherAdapter: Error during request to %s: Status=%d, Error=%v", r.Request.URL, r.StatusCode, err)
	})
	c.OnRequest(func(r *colly.Request) {
		log.Printf("KufarFetcherAdapter: Making request to %s", r.URL.String())
	})

	return &KufarFetcherAdapter{
		collector: c,
		baseURL:   baseURL,
	}
}