package usecase

import (
	"context"
	"fmt"
	"log"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"
)

// SavePropertyUseCase инкапсулирует логику сохранения RealEstateRecord
type SavePropertyUseCase struct {
	storage port.PropertyStoragePort
}

// NewSavePropertyUseCase создает новый экземпляр use case
func NewSavePropertyUseCase(storage port.PropertyStoragePort) *SavePropertyUseCase {
	return &SavePropertyUseCase{
		storage: storage,
	}
}

// сохраняет запись, используя порт хранилища
// func (uc *SavePropertyUseCase) Save(ctx context.Context, record domain.RealEstateRecord) error {
// 	log.Printf("SavePropertyUseCase: Attempting to save record from source %s\n", record.General.Source)

// 	if err := uc.storage.Save(ctx, record); err != nil {
// 		return fmt.Errorf("failed to save property record from source %s: %w", record.General.Source, err)
// 	}

// 	log.Printf("SavePropertyUseCase: Successfully saved record from source %s\n", record.General.Source)
// 	return nil
// }

func (uc *SavePropertyUseCase) BatchSave(ctx context.Context, records []domain.RealEstateRecord) error {
	log.Printf("SavePropertyUseCase: Attempting to save %d property records\n", len(records))

	if err := uc.storage.BatchSave(ctx, records); err != nil {
		return fmt.Errorf("failed to save %d property records: %w", len(records), err)
	}

	log.Printf("SavePropertyUseCase: Successfully saved %d records\n", len(records))
	return nil
}

