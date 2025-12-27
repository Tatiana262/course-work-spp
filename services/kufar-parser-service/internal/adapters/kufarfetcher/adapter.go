package kufarfetcher

import (
	"fmt"
	// "log"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
)

// KufarFetcherAdapter отвечает за все взаимодействия с сайтом Kufar
type KufarFetcherAdapter struct {
	// родительский коллектор, который разделяет лимиты
	collector *colly.Collector
	baseURL   string
}

// NewKufarFetcherAdapter - конструктор
func NewKufarFetcherAdapter(baseURL string) (*KufarFetcherAdapter, error) {
	

	// родительский коллектор
	c := colly.NewCollector(colly.AllowedDomains("api.kufar.by"), colly.AllowURLRevisit())

	// Эти правила будут наследоваться всеми клонами коллектора
	err := c.Limit(&colly.LimitRule{
		// Правило будет применяться только к API домену Kufar
		DomainGlob: "api.kufar.by",

		// Параллелизм на уровне HTTP-запросов
		Parallelism: 1,

		// задержка от 0 до 3 секунд после завершения предыдущего
		RandomDelay: 3 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("KufarFetcherAdapter: failed to set limit rule: %w", err)
	}

	extensions.RandomUserAgent(c) // На каждый запрос будет подставлен User-Agent реального браузера
	extensions.Referer(c)         // Автоматически подставляет заголовок Referer, имитируя навигацию


	return &KufarFetcherAdapter{
		collector: c,
		baseURL:   baseURL,
	}, nil
}