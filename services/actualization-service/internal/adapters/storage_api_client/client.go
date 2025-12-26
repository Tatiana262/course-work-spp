package storage_api_client

import (
	"actualization-service/internal/contextkeys"
	"actualization-service/internal/core/domain"
	"actualization-service/internal/core/port"
	"context"
	"encoding/json"
	"fmt"
	"io"

	// "log"
	"net/http"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// doRequest - внутренний хелпер для выполнения запросов
func (c *Client) doRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	// 1. Извлекаем trace_id из контекста
	traceID := contextkeys.TraceIDFromContext(ctx)

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 2. Устанавливаем заголовок для трассировки
	if traceID != "" {
		req.Header.Set("X-Trace-ID", traceID)
	}

	// Можно добавить и другие общие заголовки
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}


func (c *Client) GetCategories(ctx context.Context) ([]domain.DictionaryItem, error) {
	// 1. Извлекаем и обогащаем логгер
	logger := contextkeys.LoggerFromContext(ctx)
	clientLogger := logger.WithFields(port.Fields{
		"component": "StorageApiClient",
		"method":    "GetCategories",
	})

	url := fmt.Sprintf("%s/api/v1/dictionaries?names=categories", c.baseURL)
	clientLogger.Debug("Sending request to storage-service", port.Fields{"url": url})

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		clientLogger.Error("Failed to perform request to storage-service", err, nil)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("storage service returned non-success status code %d: %s", resp.StatusCode, string(bodyBytes))
		clientLogger.Error("Received error response from storage-service", err, port.Fields{"status_code": resp.StatusCode})
		return nil, err
	}

	var dictionaries DictionaryItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&dictionaries); err != nil {
		clientLogger.Error("Failed to decode response from storage-service", err, nil)
		return nil, err
	}

	clientLogger.Info("Successfully received and decoded response", port.Fields{"categories_count": len(dictionaries["categories"])})

	// Маппим DTO ответа в нашу доменную модель
	// Это важный шаг, который изолирует наше ядро от деталей API другого сервиса.
	result := make([]domain.DictionaryItem, len(dictionaries["categories"]))
	for i, category := range dictionaries["categories"] {
		result[i] = domain.DictionaryItem{
			SystemName: category.SystemName,
			DisplayName: category.DisplayName,
		}
	}

	return result, nil
}

func (c *Client) GetActiveObjects(ctx context.Context, category string, limit int) ([]domain.PropertyInfo, error) {

	// 1. Извлекаем и обогащаем логгер
	logger := contextkeys.LoggerFromContext(ctx)
	clientLogger := logger.WithFields(port.Fields{
		"component": "StorageApiClient",
		"method":    "GetActiveObjects",
	})

	url := fmt.Sprintf("%s/api/v1/active-objects?category=%s&limit=%d", c.baseURL, category, limit)
	clientLogger.Debug("Sending request to storage-service", port.Fields{"url": url})

	// Используем наш новый хелпер
	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		clientLogger.Error("Failed to perform request to storage-service", err, nil)
		return nil, err
	}
	defer resp.Body.Close()

	// req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

	// resp, err := c.httpClient.Do(req)
	// if err != nil {
	// 	return nil, err
	// }
	// defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Если статус-код указывает на ошибку, читаем тело ответа,
		// чтобы включить его в нашу ошибку.
		bodyBytes, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("storage service returned non-success status code %d: %s", resp.StatusCode, string(bodyBytes))
		clientLogger.Error("Received error response from storage-service", err, port.Fields{"status_code": resp.StatusCode})
		return nil, err
	}

	var objects []PropertyInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&objects); err != nil {
		clientLogger.Error("Failed to decode response from storage-service", err, nil)
		return nil, err
	}

	clientLogger.Info("Successfully received and decoded response", port.Fields{"objects_count": len(objects)})

	// 6. Маппим DTO ответа в нашу доменную модель
	// Это важный шаг, который изолирует наше ядро от деталей API другого сервиса.
	result := make([]domain.PropertyInfo, len(objects))
	for i, dto := range objects {
		result[i] = domain.PropertyInfo{
			Source: dto.Source,
			AdID:   dto.AdID,
			Link:   dto.AdLink,
		}
	}

	return result, nil
}

func (c *Client) GetArchivedObjects(ctx context.Context, category string, limit int) ([]domain.PropertyInfo, error) {

	logger := contextkeys.LoggerFromContext(ctx)
	clientLogger := logger.WithFields(port.Fields{
		"component": "StorageApiClient",
		"method":    "GetArchivedObjects",
	})

	url := fmt.Sprintf("%s/api/v1/archived-objects?category=%s&limit=%d", c.baseURL, category, limit)
	clientLogger.Debug("Sending request to storage-service", port.Fields{"url": url})

	// Используем наш новый хелпер
	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		clientLogger.Error("Failed to perform request to storage-service", err, nil)
		return nil, err
	}
	defer resp.Body.Close()

	// req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

	// resp, err := c.httpClient.Do(req)
	// if err != nil {
	// 	return nil, err
	// }
	// defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Если статус-код указывает на ошибку, читаем тело ответа,
		// чтобы включить его в нашу ошибку.
		bodyBytes, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("storage service returned non-success status code %d: %s", resp.StatusCode, string(bodyBytes))
		clientLogger.Error("Received error response from storage-service", err, port.Fields{"status_code": resp.StatusCode})
		return nil, err
	}

	var objects []PropertyInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&objects); err != nil {
		clientLogger.Error("Failed to decode response from storage-service", err, nil)
		return nil, err
	}

	clientLogger.Info("Successfully received and decoded response", port.Fields{"objects_count": len(objects)})

	// 6. Маппим DTO ответа в нашу доменную модель
	// Это важный шаг, который изолирует наше ядро от деталей API другого сервиса.
	result := make([]domain.PropertyInfo, len(objects))
	for i, dto := range objects {
		result[i] = domain.PropertyInfo{
			Source: dto.Source,
			AdID:   dto.AdID,
			Link:   dto.AdLink,
		}
	}

	return result, nil

}

func (c *Client) GetObjectsByMasterID(ctx context.Context, master_id string) ([]domain.PropertyInfo, error) {

	logger := contextkeys.LoggerFromContext(ctx)
	clientLogger := logger.WithFields(port.Fields{
		"component": "StorageApiClient",
		"method":    "GetObjectByID",
	})

	url := fmt.Sprintf("%s/api/v1/object?id=%s", c.baseURL, master_id)
	clientLogger.Debug("Sending request to storage-service", port.Fields{"url": url})

	// Используем наш новый хелпер
	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		clientLogger.Error("Failed to perform request to storage-service", err, nil)
		return nil, err
	}
	defer resp.Body.Close()

	// req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

	// resp, err := c.httpClient.Do(req)
	// if err != nil {
	// 	return nil, err
	// }
	// defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("storage service returned non-success status code %d: %s", resp.StatusCode, string(bodyBytes))
		clientLogger.Error("Received error response from storage-service", err, port.Fields{"status_code": resp.StatusCode})
		return nil, err
	}

	var objects []PropertyInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&objects); err != nil {
		clientLogger.Error("Failed to decode response from storage-service", err, nil)
		return nil, err
	}

	clientLogger.Info("Successfully received and decoded response", port.Fields{"objects_count": len(objects)})

	// 6. Маппим DTO ответа в нашу доменную модель
	// Это важный шаг, который изолирует наше ядро от деталей API другого сервиса.
	result := make([]domain.PropertyInfo, len(objects))
	for i, dto := range objects {
		result[i] = domain.PropertyInfo{
			Source: dto.Source,
			AdID:   dto.AdID,
			Link:   dto.AdLink,
		}
	}

	return result, nil
}
