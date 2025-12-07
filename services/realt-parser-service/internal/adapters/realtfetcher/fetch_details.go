package realtfetcher

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	// "encoding/json"
	// "net/url"
	// "os"
	// "path"
	"realt-parser-service/internal/contextkeys"
	"realt-parser-service/internal/core/domain"
	"realt-parser-service/internal/core/port"

	// "strings"

	"github.com/gocolly/colly/v2"
)

// --- Структуры для парсинга __NEXT_DATA__ ---

// type NextData struct {
// 	Props Props `json:"props"`
// }

// type Props struct {
// 	PageProps PageProps `json:"pageProps"`
// }

// type PageProps struct {
// 	InitialState InitialState `json:"initialState"`
// }

// type InitialState struct {
// 	ObjectView ObjectView `json:"objectView"`
// }

// type ObjectView struct {
// 	Object interface{} `json:"object"`
// }



func (a *RealtFetcherAdapter) FetchAdDetails(ctx context.Context, adURL string, adID int64) (*domain.RealEstateRecord, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	fetchDetailsLogger := logger.WithFields(port.Fields{"component": "RealtFetcherAdapter(FetchDetails)"})
	
	collector := a.collector.Clone()

	// var parsedData kufarAdViewData
	var record *domain.RealEstateRecord
	var criticalError error
	var rawJson string

	collector.OnHTML("script#__NEXT_DATA__", func(e *colly.HTMLElement) {	
		rawJson = e.Text
	})

	collector.OnError(func(r *colly.Response, err error) {
		fetchDetailsLogger.Error("Failed to fetch ad details", err, port.Fields{
			"ad_id":  adID,
			"url":    r.Request.URL.String(),
			"status": r.StatusCode,
		})

		if r.StatusCode == http.StatusNotFound || r.StatusCode == http.StatusGone {
			fetchDetailsLogger.Warn("Ad is not available (404/410), creating archive record", port.Fields{"ad_id": adID})
			// Создаем "пустой" объект-заглушку для архивации
			record = &domain.RealEstateRecord{
				General: domain.GeneralProperty{
					Source:     "realt",
					SourceAdID: adID,
					Status:     domain.StatusArchived,
				},
			}
			// Важно! Не устанавливаем `criticalError`, так как мы успешно обработали этот случай.
			return 
		}

        criticalError = fmt.Errorf("FetchAdDetails: colly error on %s: status %d: %w", adURL, r.StatusCode, err) // Сохраняем ошибку для возврата
	})

	// Этот колбэк вызывается после того, как все OnHTML отработали
	collector.OnScraped(func(r *colly.Response) {

		if criticalError != nil || record != nil {
			return
		}
		
		if rawJson == "" {
			fetchDetailsLogger.Warn("Could not find JSON data on the page, but status was not bad", port.Fields{"ad_id": adID})
			criticalError = errors.New("could not find JSON data on the page, but status was not 404")
			return
		}

		
		rec, err := toDomainRecord(rawJson, adURL, "realt", fetchDetailsLogger)
		if err != nil {
			fetchDetailsLogger.Error("Failed to map response to domain record", err, port.Fields{"ad_id": adID})
			criticalError = fmt.Errorf("FetchAdDetails: failed to map response to domain record: %w", err)
			return
		}
		record = rec
		
	})

	_ = collector.Visit(adURL)
	// if visitErr != nil {
	// 	return nil, fmt.Errorf("FetchAdDetails: failed to visit URL %s: %w", adURL, visitErr)
	// }
	collector.Wait() // Ждем завершения HTTP запроса и выполнения OnHTML

	return record, criticalError
}





// var data NextData
		// err := json.Unmarshal([]byte(rawJson), &data)
		// if err != nil {
		// 	log.Fatalf("Ошибка парсинга JSON: %v", err)
		// }

		// obj := data.Props.PageProps.InitialState.ObjectView.Object

		// objectJson, err := json.Marshal(obj)
		// if err != nil {
		// 	log.Printf("Не удалось сериализовать 'object' обратно в JSON: %v", err)
		// 	return // или fetchErr = err
		// }

		// // Создаем папку, если ее нет
		// _ = os.MkdirAll("api_responses_2/proizvodstvo_brest_rent", 0755)
				
		// u, err := url.Parse(adURL)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// last := path.Base(u.Path)
		// // Формируем имя файла на основе ad_id, чтобы избежать дубликатов
		// filename := fmt.Sprintf("api_responses_2/proizvodstvo_brest_rent/%s.json", last)

		// // Записываем "сырое" тело ответа в файл
		// err = os.WriteFile(filename, objectJson, 0644)
		// if err != nil {
		// 	log.Printf("Failed to save response for adURL %s: %v", adURL, err)
		// } else {
		// 	log.Printf("Successfully saved response for adURL %s to %s", adURL, filename)
		// }




		// // Создаем папку, если ее нет
		// _ = os.MkdirAll("json_objects/", 0755)
				
		// u, err := url.Parse(adURL)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// last := path.Base(u.Path)
		// // Формируем имя файла на основе ad_id, чтобы избежать дубликатов
		// filename := fmt.Sprintf("json_objects/%s.json", last)

		// objectJson, err := json.Marshal(record)
		// if err != nil {
		// 	log.Fatalf("Ошибка маршаллинга JSON: %v", err)
		// }

		// // Записываем "сырое" тело ответа в файл
		// err = os.WriteFile(filename, objectJson, 0644)
		// if err != nil {
		// 	log.Printf("Failed to save response for adURL %s: %v", adURL, err)
		// } else {
		// 	log.Printf("Successfully saved response for adURL %s to %s", adURL, filename)
		// }