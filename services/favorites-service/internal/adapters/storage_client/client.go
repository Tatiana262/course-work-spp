package storage_api_client

import (
	"bytes"
	"context"
	"encoding/json"
	"favorites-service/internal/contextkeys"
	"favorites-service/internal/core/domain"
	"favorites-service/internal/core/port"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

// StorageServiceAPIClient - клиент для взаимодействия с storage-service.
type StorageServiceAPIClient struct {
	baseURL    string       // Например, "http://storage-service:8080"
	httpClient *http.Client
}



// NewStorageServiceAPIClient - конструктор.
func NewStorageServiceAPIClient(baseURL string) *StorageServiceAPIClient {
	return &StorageServiceAPIClient{
		baseURL: baseURL,
		httpClient: &http.Client{},
	}
}

// doRequest - внутренний хелпер для выполнения запросов
func (c *StorageServiceAPIClient) doRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
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

// GetBestObjectsByMasterIDs реализует порт ObjectStoragePort.
func (c *StorageServiceAPIClient) GetBestObjectsByMasterIDs(ctx context.Context, masterIDs []uuid.UUID) ([]domain.ObjectCard, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	clientLogger := logger.WithFields(port.Fields{
		"component":       "StorageServiceAPIClient",
		"method":          "GetBestObjectsByMasterIDs",
		"master_id_count": len(masterIDs),
	})
	
	if len(masterIDs) == 0 {
		clientLogger.Info("Received empty list of master IDs, returning empty result.", nil)
		return []domain.ObjectCard{}, nil
	}

	// 1. Формируем тело запроса
	reqBody, err := json.Marshal(getByMasterIDsRequest{MasterIDs: masterIDs})
	if err != nil {
		clientLogger.Error("Failed to marshal request body", err, nil)
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 2. Создаем HTTP POST-запрос
	url := c.baseURL + "/api/v1/objects/best-by-master-ids"
	resp, err := c.doRequest(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		clientLogger.Error("Failed to perform request to storage-service", err, nil)
		return nil, err
	}
	defer resp.Body.Close()


	// req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	// if err != nil {
	// 	clientLogger.Error("Failed to create request object", err, nil)
	// 	return nil, fmt.Errorf("failed to create request to storage service: %w", err)
	// }

	// req.Header.Set("Content-Type", "application/json")
	// traceID := contextkeys.TraceIDFromContext(ctx)
	// if traceID != "" {
	// 	req.Header.Set("X-Trace-ID", traceID) // Используем X-Trace-ID для единообразия
	// }

	// clientLogger.Info("Sending request to storage-service.", port.Fields{"url": url})

	// // 3. Выполняем запрос
	// resp, err := c.httpClient.Do(req)
	// if err != nil {
	// 	clientLogger.Error("Failed to execute request to storage service", err, nil)
	// 	return nil, fmt.Errorf("failed to execute request to storage service: %w", err)
	// }
	// defer resp.Body.Close()

	// 4. Проверяем статус-код ответа
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("storage service returned non-200 status: %d, body: %s", resp.StatusCode, string(bodyBytes))
		clientLogger.Error("Received non-OK response from storage service", err, port.Fields{"status_code": resp.StatusCode})
		return nil, err
	}

	// 5. Декодируем тело ответа
	// Ответ от storage-service может быть в виде {"data": [...]}, нужно это учесть.
	var apiResponse getByMasterIDsResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		clientLogger.Error("Failed to decode response from storage service", err, nil)
		return nil, fmt.Errorf("failed to decode response from storage service: %w", err)
	}
	clientLogger.Info("Successfully received and decoded response from storage service.", port.Fields{"objects_found": len(apiResponse.Data)})
	
	// 6. Маппим DTO ответа в нашу доменную модель
	// Это важный шаг, который изолирует наше ядро от деталей API другого сервиса.
	result := make([]domain.ObjectCard, len(apiResponse.Data))
	for i, dto := range apiResponse.Data {
		result[i] = domain.ObjectCard{
			ID:             dto.ID,
			MasterObjectID: dto.MasterObjectID,
			Title:          dto.Title,
			PriceUSD:       dto.PriceUSD,
			Images:         dto.Images,
			Address:        dto.Address,
			Status:         dto.Status,
		}
	}

	return result, nil
}