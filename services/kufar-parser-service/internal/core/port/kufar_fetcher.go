package port

import (
	"context"
	"kufar-parser-service/internal/core/domain"
	"time"
)

// KufarFetcherPort объединяет все операции, которые можно выполнить
// с источником данных Kufar.
type KufarFetcherPort interface {
	// FetchLinks извлекает ссылки, соответствующие критериям.
	FetchLinks(ctx context.Context, criteria domain.SearchCriteria, since time.Time) (links []domain.PropertyLink, nextCursor string, err error)
	
	// FetchAdDetails извлекает полную информацию об объекте недвижимости по его URL.
	FetchAdDetails(ctx context.Context, adId int64) (*domain.RealEstateRecord, error)
}