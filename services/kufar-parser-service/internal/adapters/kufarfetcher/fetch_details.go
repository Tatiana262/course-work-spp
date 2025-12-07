package kufarfetcher

import (
	"context"
	// "encoding/json"
	"fmt"
	// "log"
	"net/http"

	// "os"
	"kufar-parser-service/internal/contextkeys"
	"kufar-parser-service/internal/core/domain"
	"kufar-parser-service/internal/core/port"

	// "regexp"
	// "strconv"
	// "strings"

	// "time"

	"github.com/gocolly/colly/v2"
)

// FetchAdDetails извлекает и преобразует детальную информацию об объявлении
func (a *KufarFetcherAdapter) FetchAdDetails(ctx context.Context, adID int64) (*domain.RealEstateRecord, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	fetchDetailsLogger := logger.WithFields(port.Fields{"component": "KufarFetcherAdapter(FetchDetails)"})

	collector := a.collector.Clone()

	// var parsedData kufarAdViewData
	var record *domain.RealEstateRecord
	var criticalError error

	collector.OnRequest(func(r *colly.Request) {
		fetchDetailsLogger.Info("Making request to fetch ad details", port.Fields{
			"url":   r.URL.String(),
			"ad_id": adID,
		})
	})

	// OnResponse сработает, когда мы получим успешный ответ от API.
	collector.OnResponse(func(r *colly.Response) {

		if criticalError != nil || record != nil {
			return
		}

		rec, err := toDomainRecord(r.Body, "kufar", fetchDetailsLogger)
		if err != nil {
			fetchDetailsLogger.Error("Failed to map response to domain record", err, port.Fields{"ad_id": adID})
			criticalError = fmt.Errorf("FetchAdDetails: failed to map response to domain record: %w", err)
			return
		}
		record = rec
		
	})

	 // Этот колбэк будет вызван для ошибок, специфичных для этого запроса
	collector.OnError(func(r *colly.Response, err error) {

		fetchDetailsLogger.Error("Failed to fetch ad details", err, port.Fields{
			"ad_id":  adID,
			"url":    r.Request.URL.String(),
			"status": r.StatusCode,
		})

		// Если страница не найдена (404) или удалена (410), это не ошибка парсинга.
		// Это информация о том, что объявление нужно архивировать.
		if r.StatusCode == http.StatusNotFound || r.StatusCode == http.StatusGone {
			fetchDetailsLogger.Warn("Ad is not available (404/410), creating archive record", port.Fields{"ad_id": adID})
			record = &domain.RealEstateRecord{
				General: domain.GeneralProperty{
					Source:     "kufar",
					SourceAdID: adID,
					Status:     domain.StatusArchived,
				},
			}
			// Важно! Не устанавливаем `criticalError`, так как мы успешно обработали этот случай.
			return 
		}

		criticalError = fmt.Errorf("FetchAdDetails: colly error on %d: status %d: %w", adID, r.StatusCode, err)
    })

	// Формируем URL для API, используя adID
	apiURL := fmt.Sprintf("https://api.kufar.by/search-api/v2/item/%d/rendered", adID)
	_ = collector.Visit(apiURL)
	// if visitErr != nil {
	// 	return nil, fmt.Errorf("kufar adapter (Detail): failed to visit URL %s: %w", apiURL, visitErr)
	// }
	collector.Wait() // Ждем завершения HTTP запроса и выполнения OnHTML

	return record, criticalError
}




// // Создаем папку, если ее нет
        // _ = os.MkdirAll("api_responses_2", 0755)
        
        // // Формируем имя файла на основе ad_id, чтобы избежать дубликатов
        // filename := fmt.Sprintf("api_responses_2/ad_%d.json", adID)
        
        // // Записываем "сырое" тело ответа в файл
        // err := os.WriteFile(filename, r.Body, 0644)
        // if err != nil {
        //     log.Printf("Failed to save response for ad_id %d: %v", adID, err)
        // } else {
        //     log.Printf("Successfully saved response for ad_id %d to %s", adID, filename)
        // }


// // Вспомогательная функция для получения указателя на строку
// func strPtr(s string) *string {
// 	return &s
// }

// // Вспомогательная функция для очистки строки с ценой
// func cleanPriceString(priceStr string) float64 {
// 	// Регулярное выражение для удаления всего, кроме цифр и точки/запятой
// 	re := regexp.MustCompile(`[^\d.,]`)
// 	cleaned := re.ReplaceAllString(priceStr, "")
// 	// Заменяем запятую на точку для корректного парсинга
// 	cleaned = strings.Replace(cleaned, ",", ".", -1)

// 	price, err := strconv.ParseFloat(cleaned, 64)
// 	if err != nil {
// 		return 0
// 	}
// 	return price
// }


// // Вспомогательная функция для преобразования interface{} в строку
// // Вспомогательная функция для преобразования interface{} в строку для вывода
// func valueToString(val interface{}) string {
// 	if val == nil {
// 		return "(null)"
// 	}
// 	switch v := val.(type) {
// 	case string:
// 		if v == "" || v == "-" { // Пустую строку или дефис считаем "незначимым" для vl
// 			return "" // Вернем пустую строку, чтобы потом взять значение из 'V'
// 		}
// 		return v
// 	case float64:
// 		return strconv.FormatFloat(v, 'f', -1, 64)
// 	case bool:
// 		return strconv.FormatBool(v)
// 	case []interface{}: // Для массивов в vl или v
// 		var strVals []string
//         nonEmptyCount := 0
// 		for _, item := range v {
//             itemStr := valueToString(item) // Рекурсивный вызов для элементов массива
//             if itemStr != "" && itemStr != "(null)" { // Собираем только непустые
// 			    strVals = append(strVals, itemStr)
//                 nonEmptyCount++
//             }
// 		}
//         if nonEmptyCount > 0 {
// 		    return strings.Join(strVals, ", ") // Не добавляем скобки здесь, т.к. это значение для отображения
//         }
//         return "" // Если массив пуст или содержит только пустые значения
// 	default:
// 		// Для json.Number (если числа приходят так из-за AdID) или других непредвиденных типов
// 		if num, ok := val.(json.Number); ok {
// 			return num.String()
// 		}
// 		return fmt.Sprintf("%v", v) // Общий случай
// 	}
// }




// type kufarDetailRoot struct {
// 	Result kufarAdResult `json:"result"`
// }

// type kufarAdResult struct {
// 	AdID          int          `json:"ad_id"` // json.Number здесь хорошо, если ID может быть большим
// 	AdURL		  string		  `json:"ad_link"`
// 	Subject       string               `json:"subject"`
// 	Body          string               `json:"body"`

// 	PriceBYN      string               `json:"price_byn"`    // Если это всегда строка в JSON
// 	PriceUSD      string               `json:"price_usd"` // Если это всегда строка в JSON
// 	Currency      string               `json:"currency"`

// 	ListTime      string               `json:"list_time"`
// 	Images        []kufarImage         `json:"images"`

// 	IsCompanyAd   bool                 `json:"company_ad"`


// 	AdParams []kufarAdParameterItem `json:"ad_parameters"`
// 	AccountParams []kufarAccountParameterItem `json:"account_parameters"`
// }

// type kufarAdParameterItem struct {
// 	Label       string            `json:"pl"`
// 	ValueString interface{}       `json:"vl"` // Используем interface{}
// 	P           string            `json:"p"`
// 	V           interface{}       `json:"v"`  // Используем interface{}
// 	Pu          string            `json:"pu"`
// 	G           []kufarParamGroup `json:"g,omitempty"` // Добавил omitempty, т.к. g не везде есть
// }

// type kufarAccountParameterItem struct {
// 	Label string            `json:"pl"`
// 	Value string            `json:"v"`
// 	P     string            `json:"p"`
// 	Pu    string            `json:"pu"`
// 	G     []kufarParamGroup `json:"g,omitempty"`
// }

// type kufarParamGroup struct {
// 	GroupID    int    `json:"gi"`
// 	GroupLabel string `json:"gl"`
// 	GroupOrder int    `json:"go"`
// 	ParamOrder int    `json:"po"`
// }

// type kufarImage struct {
// 	Path string `json:"path"`
// }