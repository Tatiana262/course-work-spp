package usecase

import (
	"context"
	// "encoding/json"
	"fmt"
	// "log"
	// "os"
	// "path/filepath"

	// "fmt"
	// "log"
	"realt-parser-service/internal/contextkeys"
	"realt-parser-service/internal/core/domain"
	"realt-parser-service/internal/core/port"

	"github.com/google/uuid"
)

// ProcessLinkUseCase инкапсулирует логику обработки одной ссылки:
// парсинг деталей и отправка результата в следующую очередь.
type ProcessLinkUseCase struct {
	detailsFetcher port.RealtFetcherPort
	resultQueue    port.ProcessedPropertyQueuePort
}

// NewProcessLinkUseCase создает новый экземпляр use case.
func NewProcessLinkUseCase(
	fetcher port.RealtFetcherPort,
	queue port.ProcessedPropertyQueuePort,
) *ProcessLinkUseCase {
	return &ProcessLinkUseCase{
		detailsFetcher: fetcher,
		resultQueue:    queue,
	}
}

// 
// Execute выполняет основную логику use case.
func (uc *ProcessLinkUseCase) Execute(ctx context.Context, linkToParse domain.PropertyLink, taskID uuid.UUID) error {

	// 1. Извлекаем и обогащаем логгер
	baseLogger := contextkeys.LoggerFromContext(ctx)
	ucLogger := baseLogger.WithFields(port.Fields{
		"use_case": "ProcessLink",
		// "ad_id":    linkToParse.AdID,
		// "task_id":  taskID,
	})
	
	ucLogger.Info("Processing link", nil)

	// 1. Используем порт для парсинга деталей
	propertyRecord, fetchErr := uc.detailsFetcher.FetchAdDetails(ctx, linkToParse.URL, linkToParse.AdID)
	if fetchErr != nil {
		ucLogger.Error("Failed to fetch/parse details", fetchErr, nil)
		// Ошибка возвращается наверх, чтобы обработчик RabbitMQ мог решить, что делать (requeue/nack)
		return fmt.Errorf("failed to fetch/parse details for %d: %w", linkToParse.AdID, fetchErr)
	}

	if propertyRecord.General.Status == domain.StatusArchived {
		ucLogger.Info("Successfully processed as archived.", nil)
	} else {
		ucLogger.Info("Successfully parsed details.", port.Fields{"title": propertyRecord.General.Title})
	}

	// 2. Используем порт для отправки результата в очередь
	err := uc.resultQueue.Enqueue(ctx, *propertyRecord, taskID)
	if err != nil {
		ucLogger.Error("failed to enqueue processed data", err, nil)
		return fmt.Errorf("CRITICAL: failed to enqueue processed data for AdID %d: %w", linkToParse.AdID, err)
	}

	ucLogger.Info("Successfully enqueued processed data", nil)
	return nil
}


// func appendJSONToFile(dir, filename string, data interface{}) error {
	// 	// 1️⃣ Создаём директорию, если её нет
	// 	if err := os.MkdirAll(dir, 0755); err != nil {
	// 		return fmt.Errorf("failed to create directory: %w", err)
	// 	}
	
	// 	// 2️⃣ Полный путь к файлу
	// 	filePath := filepath.Join(dir, filename)
	
	// 	// 3️⃣ Открываем файл на дозапись (создаём, если не существует)
	// 	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to open file: %w", err)
	// 	}
	// 	defer f.Close()
	
	// 	// 4️⃣ Кодируем данные в JSON
	// 	jsonBytes, err := json.Marshal(data)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to marshal json: %w", err)
	// 	}
	
	// 	// 5️⃣ Пишем JSON и перевод строки для читаемости
	// 	if _, err := f.Write(append(jsonBytes, '\n')); err != nil {
	// 		return fmt.Errorf("failed to write json: %w", err)
	// 	}
	
	// 	return nil
	// }
	