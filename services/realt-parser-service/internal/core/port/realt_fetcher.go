package port

import (
	"context"
	"realt-parser-service/internal/core/domain"
	"time"
)


type RealtFetcherPort interface {
	// FetchLinks извлекает ссылки, соответствующие критериям.
	FetchLinks(ctx context.Context, criteria domain.SearchCriteria, since time.Time) (links []domain.PropertyLink, nextPage int, err error)
	
	// FetchAdDetails извлекает полную информацию об объекте недвижимости по его URL.
	FetchAdDetails(ctx context.Context, adURL string, adID int64) (*domain.RealEstateRecord, error)
}