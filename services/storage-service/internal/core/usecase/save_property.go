package usecase

import (
	"context"
	"fmt"
	// "log"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"

	"github.com/google/uuid"
)

// SavePropertyUseCase инкапсулирует логику сохранения PropertyRecord.
type SavePropertyUseCase struct {
	storage port.PropertyStoragePort
	reporter port.TaskReporterPort
}

// NewSavePropertyUseCase создает новый экземпляр use case.
func NewSavePropertyUseCase(storage port.PropertyStoragePort, reporter port.TaskReporterPort) *SavePropertyUseCase {
	return &SavePropertyUseCase{
		storage: storage,
		reporter: reporter,
	}
}

// Execute выполняет основную логику: сохраняет запись, используя порт хранилища.
func (uc *SavePropertyUseCase) Save(ctx context.Context, record domain.RealEstateRecord) error {
	
	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case": "SaveProperty",
		"source":   record.General.Source,
		"ad_id":    record.General.SourceAdID,
	})
	
	ucLogger.Info("Use case started: attempting to save single record", nil)

	if err := uc.storage.Save(ctx, record); err != nil {
		ucLogger.Error("Storage returned an error during save", err, nil)
		return fmt.Errorf("failed to save property record from source %s: %w", record.General.Source, err)
	}

	ucLogger.Info("Use case finished: successfully saved single record", nil)
	return nil
}

func (uc *SavePropertyUseCase) BatchSave(ctx context.Context, records []domain.RealEstateRecord, taskID uuid.UUID) error {
	logger := contextkeys.LoggerFromContext(ctx)
	ucLogger := logger.WithFields(port.Fields{
		"use_case":     "BatchSaveProperty",
		"record_count": len(records),
		"task_id":      taskID.String(),
	})
	
	ucLogger.Info("Use case started: attempting to batch save records", nil)

	stats, err := uc.storage.BatchSave(ctx, records)
    if err != nil {
		ucLogger.Error("Storage returned an error during batch save", err, nil)
        return fmt.Errorf("failed to save %d property records: %w", len(records), err)
    }

	ucLogger.Info("Storage batch save completed successfully", port.Fields{"stats": stats})

	// 2. Если статистика не пустая, отправляем отчет
    if stats != nil && (stats.Created > 0 || stats.Updated > 0 || stats.Archived > 0) {
        if err := uc.reporter.ReportResults(ctx, taskID, stats); err != nil {
            // Логируем ошибку, но не возвращаем ее, т.к. основная операция (сохранение) прошла успешно.
            // Это предотвратит повторную обработку уже сохраненных данных.
			ucLogger.Error("Failed to report task results after successful save", err, nil)
        }
    }

	ucLogger.Info("Use case finished", nil)
	return nil
}

