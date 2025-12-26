package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	// "log"

	// "os"

	"storage-service/internal/contextkeys"
	"storage-service/internal/contracts"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"
	usecases_port "storage-service/internal/core/port/usecases_port"

	// "storage-service/internal/core/usecase"
	"real-estate-system/pkg/rabbitmq/rabbitmq_common"
	"real-estate-system/pkg/rabbitmq/rabbitmq_consumer"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// DetailUnmarshaler определяет контракт для сущности, которая может
// десериализовать определенный тип деталей из JSON.
type DetailUnmarshaler interface {
	UnmarshalDetails(data json.RawMessage) (interface{}, error)
}

// ApartmentUnmarshaler реализует интерфейс для деталей квартиры.
type ApartmentUnmarshaler struct{}

func (u *ApartmentUnmarshaler) UnmarshalDetails(data json.RawMessage) (interface{}, error) {
	var detailsDTO ApartmentDetailsDTO
	if err := json.Unmarshal(data, &detailsDTO); err != nil {
		return nil, err
	}
	return toDomainApartment(&detailsDTO), nil
}

// HouseUnmarshaler реализует интерфейс для деталей дома.
type HouseUnmarshaler struct{}

func (u *HouseUnmarshaler) UnmarshalDetails(data json.RawMessage) (interface{}, error) {
	var detailsDTO HouseDetailsDTO
	if err := json.Unmarshal(data, &detailsDTO); err != nil {
		return nil, err
	}
	return toDomainHouse(&detailsDTO), nil
}

type CommercialUnmarshaler struct{}

func (u *CommercialUnmarshaler) UnmarshalDetails(data json.RawMessage) (interface{}, error) {
	var detailsDTO CommercialDetailsDTO
	if err := json.Unmarshal(data, &detailsDTO); err != nil {
		return nil, err
	}
	return toDomainCommercial(&detailsDTO), nil
}

// GenericUnmarshaler для типов, не требующих DTO-трансляции
type GenericUnmarshaler[T any] struct{}

func (u *GenericUnmarshaler[T]) UnmarshalDetails(data json.RawMessage) (interface{}, error) {
	var details T
	if err := json.Unmarshal(data, &details); err != nil {
		return nil, err
	}
	return &details, nil
}

// ProcessedPropertyConsumerAdapter - это входящий адаптер, который слушает очередь
// с обработанными объектами недвижимости и вызывает use case для их сохранения.
type ProcessedPropertyConsumerAdapter struct {
	consumer        rabbitmq_consumer.Consumer
	useCase         usecases_port.SavePropertyPort // Зависимость от конкретного UseCase
	logger          port.LoggerPort
	detailsRegistry map[string]DetailUnmarshaler
}

// NewProcessedPropertyConsumerAdapter создает новый адаптер.
func NewProcessedPropertyConsumerAdapter(
	consumerCfg rabbitmq_consumer.ConsumerConfig,
	useCase usecases_port.SavePropertyPort,
	logger port.LoggerPort,
	connManager *rabbitmq_common.ConnectionManager,
) (*ProcessedPropertyConsumerAdapter, error) {

	adapter := &ProcessedPropertyConsumerAdapter{
		useCase:         useCase,
		logger:          logger,
		detailsRegistry: make(map[string]DetailUnmarshaler),
	}

	// 1. Создаем логгер для pkg-уровня с контекстом нашего компонента
	pkgLogger := logger.WithFields(port.Fields{"component": "rabbitmq_batch_consumer", "consumer_tag": consumerCfg.ConsumerTag})
	consumerCfg.Logger = NewPkgLoggerBridge(pkgLogger)

	adapter.detailsRegistry["apartment"] = &ApartmentUnmarshaler{}
	adapter.detailsRegistry["house"] = &HouseUnmarshaler{}
	adapter.detailsRegistry["commercial"] = &CommercialUnmarshaler{}
	adapter.detailsRegistry["garage_and_parking"] = &GenericUnmarshaler[domain.GarageAndParking]{}
	adapter.detailsRegistry["room"] = &GenericUnmarshaler[domain.Room]{}
	adapter.detailsRegistry["plot"] = &GenericUnmarshaler[domain.Plot]{}
	adapter.detailsRegistry["new_building"] = &GenericUnmarshaler[domain.NewBuilding]{}

	// Создаем consumer, передавая ему метод этого адаптера как обработчик.
	consumer, err := rabbitmq_consumer.NewBatchConsumer(consumerCfg, adapter.batchMessageHandler, 100, 10*time.Second, connManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ consumer for processed properties: %w", err)
	}
	adapter.consumer = consumer

	return adapter, nil
}

// batchMessageHandler - НОВЫЙ обработчик, который принимает СРЕЗ сообщений.
func (a *ProcessedPropertyConsumerAdapter) batchMessageHandler(deliveries []amqp.Delivery) error {

	if len(deliveries) == 0 {
		return nil // Пустая пачка, ничего не делаем
	}

	traceID, _ := deliveries[0].Headers["x-trace-id"].(string)
	if traceID == "" {
		// Если по какой-то причине его нет, генерируем новый.
		traceID = uuid.New().String()
	}

	// 1.2. Генерируем уникальный ID для этой конкретной операции батчинга.
	batchID := uuid.New().String()

	// 1.3. Создаем базовый логгер для всей операции. Он будет содержать ОБА ID.
	batchLogger := a.logger.WithFields(port.Fields{
		"trace_id":     traceID, // <-- СОХРАНЯЕМ СКВОЗНУЮ ТРАССИРОВКУ
		"batch_id":     batchID, // <-- Добавляем контекст батча
		"batch_size":   len(deliveries),
		"adapter_name": "ProcessedPropertyConsumerAdapter",
	})

	// 1.4. Создаем контекст и кладем в него логгер и ОСНОВНОЙ trace_id
	ctx := context.Background()
	ctx = contextkeys.ContextWithLogger(ctx, batchLogger)
	ctx = contextkeys.ContextWithTraceID(ctx, traceID) // <-- ВАЖНО: передаем дальше именно trace_id

	batchLogger.Info("Received batch of messages to process.", nil)

	recordsByTask := make(map[uuid.UUID][]domain.RealEstateRecord)

	// 1. Разбираем все сообщения в пачке
	for _, d := range deliveries {
		record, taskID, err := a.unmarshalRecord(d, batchLogger)
		if err != nil {
			// log.Printf("FATAL: Failed to unmarshal message in batch, tag %d: %v. This will cause requeue of the whole batch.", d.DeliveryTag, err)
			// // Если хотя бы одно сообщение "битое", возвращаем ошибку,
			// // чтобы вся пачка вернулась в очередь (и в итоге попала в DLX).
			return err
		}
		if record != nil {
			recordsByTask[taskID] = append(recordsByTask[taskID], *record)
		}
	}

	if len(recordsByTask) == 0 {
		batchLogger.Info("No valid records in batch to save.", nil)
		return nil
	}

	// Теперь вызываем BatchSave для каждой группы задач
	for taskID, records := range recordsByTask {
		taskLogger := batchLogger.WithFields(port.Fields{"task_id": taskID.String()})
		taskLogger.Info("Calling BatchSave for records from task...", port.Fields{"record_count": len(records)})

		// Передаем taskID в Use Case
		if err := a.useCase.BatchSave(ctx, records, taskID); err != nil {
			taskLogger.Error("BatchSave failed, the entire batch will be requeued.", err, nil)
			// Если хотя бы одна группа не сохранилась, возвращаем ошибку, чтобы весь батч обработался снова
			return err
		}
	}

	batchLogger.Info("Batch processed successfully.", nil)
	return nil

	// Сериализуем с отступами для читаемости
	// prettyJSON, err := json.MarshalIndent(records, "", "  ")
	// if err != nil {
	// 	return fmt.Errorf("failed to format record to pretty JSON for URL: %w", err)
	// }

	// // Открываем файл для дозаписи
	// file, err := os.OpenFile("data.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	// if err != nil {
	// 	return fmt.Errorf("failed to open output file '%s': %w", "data.json", err)
	// }
	// defer file.Close()

	// // Собираем полный блок для записи
	// separator := "\n\n"
	// dataToWrite := append(prettyJSON, []byte(separator)...)

	// if _, err := file.Write(dataToWrite); err != nil {
	// 	return fmt.Errorf("failed to write to output file '%s': %w", "data.json", err)
	// }
}

// unmarshalRecord - НОВАЯ функция-помощник для разбора ОДНОГО сообщения.
// Мы вынесли эту логику из старого обработчика.
func (a *ProcessedPropertyConsumerAdapter) unmarshalRecord(d amqp.Delivery, parentLogger port.LoggerPort) (*domain.RealEstateRecord, uuid.UUID, error) {
	msgLogger := parentLogger.WithFields(port.Fields{
		"message_id": d.MessageId,
		// Можно даже проверить, совпадает ли trace_id сообщения с основным
		"original_trace_id": d.Headers["x-trace-id"],
	})

	// 1. Валидация по схеме (этот код у вас уже есть)
	eventType, _ := d.Headers["event-type"].(string)
	eventVersion, _ := d.Headers["event-version"].(string)
	if err := contracts.ValidateEvent(eventType, eventVersion, d.Body); err != nil {
		msgLogger.Error("Message failed schema validation. Rejecting.", err, nil)
		return nil, uuid.Nil, err
	}

	// 2. Десериализация в DTO
	var dto IncomingEventDTO
	if err := json.Unmarshal(d.Body, &dto); err != nil {
		return nil, uuid.Nil, fmt.Errorf("failed to unmarshal incoming event DTO: %w", err)
	}

	// 3. ТРАНСЛЯЦИЯ из DTO в НОВЫЙ домен
	// Это ключевой момент. Мы преобразуем "внешние" данные во "внутренние".

	record := &domain.RealEstateRecord{
		General: toDomainGeneralProperty(dto),
	}

	unmarshaler, found := a.detailsRegistry[dto.DetailsType]
	if !found {
		// 2. Если не найден - логируем и пропускаем, не считая это фатальной ошибкой
		parentLogger.Warn("Unknown details type, details will be ignored.", port.Fields{"details_type": dto.DetailsType})
	} else {
		// 3. Если найден - вызываем его, и он сам сделает всю работу
		details, err := unmarshaler.UnmarshalDetails(dto.Details)
		if err != nil {
			parentLogger.Error("Error unmarshalling details", err, port.Fields{"details_type": dto.DetailsType})
			return nil, uuid.Nil, err
		}
		record.Details = details
	}

	return record, dto.TaskID, nil
}

func toDomainGeneralProperty(dto IncomingEventDTO) domain.GeneralProperty {

	wktPoint := fmt.Sprintf("SRID=4326;POINT(%f %f)", dto.General.Longitude, dto.General.Latitude)

	return domain.GeneralProperty{ // Маппим поля
		ID:             uuid.New(),
		Source:         dto.General.Source,
		SourceAdID:     dto.General.SourceAdID,
		UpdatedAt:      time.Now(),
		CreatedAt:      time.Now(),
		Category:       dto.DetailsType,
		AdLink:         dto.General.AdLink,
		SaleType:       dto.General.SaleType,
		Currency:       dto.General.Currency,
		Images:         dto.General.Images,
		ListTime:       dto.General.ListTime,
		Description:    dto.General.Description,
		Title:          dto.General.Title,
		DealType:       dto.General.DealType,
		Coordinates:    wktPoint,
		CityOrDistrict: dto.General.CityOrDistrict,
		Region:         dto.General.Region,
		PriceBYN:       dto.General.PriceBYN,
		PriceUSD:       dto.General.PriceUSD,
		PriceEUR:       dto.General.PriceEUR,
		Address:        dto.General.Address,

		IsAgency:      dto.General.IsAgency,
		SellerName:    dto.General.SellerName,
		SellerDetails: dto.General.SellerDetails,

		Status: dto.General.Status,

		Latitude:  dto.General.Latitude,
		Longitude: dto.General.Longitude,
	}
}

func toDomainApartment(dto *ApartmentDetailsDTO) *domain.Apartment {
	return &domain.Apartment{
		RoomsAmount:         dto.RoomsAmount,
		FloorNumber:         dto.FloorNumber,
		BuildingFloors:      dto.BuildingFloors,
		TotalArea:           dto.TotalArea,
		LivingSpaceArea:     dto.LivingSpaceArea,
		KitchenArea:         dto.KitchenArea,
		YearBuilt:           dto.YearBuilt,
		WallMaterial:        dto.WallMaterial,
		RepairState:         dto.RepairState,
		BathroomType:        dto.BathroomType,
		BalconyType:         dto.BalconyType,
		PricePerSquareMeter: dto.PricePerSquareMeter,
		IsNewCondition:		dto.IsNewCondition,
		Parameters:          dto.Parameters,
	}
}

func toDomainHouse(dto *HouseDetailsDTO) *domain.House {
	return &domain.House{
		TotalArea:         dto.TotalArea,
		PlotArea:          dto.PlotArea,
		WallMaterial:      dto.WallMaterial,
		YearBuilt:         dto.YearBuilt,
		LivingSpaceArea:   dto.LivingSpaceArea,
		BuildingFloors:    dto.BuildingFloors,
		RoomsAmount:       dto.RoomsAmount,
		KitchenArea:       dto.KitchenArea, // Имя поля в домене и DTO совпадает
		Electricity:       dto.Electricity,
		Water:             dto.Water,
		Heating:           dto.Heating,
		Sewage:            dto.Sewage,
		Gaz:               dto.Gaz,
		RoofMaterial:      dto.RoofMaterial,
		HouseType:         dto.HouseType,
		CompletionPercent: dto.CompletionPercent,
		IsNewCondition:		dto.IsNewCondition,
		Parameters:        dto.Parameters,
	}
}

func toDomainCommercial(dto *CommercialDetailsDTO) *domain.Commercial {
	return &domain.Commercial{
		IsNewCondition: dto.IsNewCondition,
		PropertyType: dto.PropertyType,
		FloorNumber: dto.FloorNumber,
		BuildingFloors: dto.BuildingFloors,
		TotalArea: dto.TotalArea,
		CommercialImprovements: dto.CommercialImprovements,
		CommercialRepair: dto.CommercialRepair,
		PricePerSquareMeter: dto.PricePerSquareMeter,
		RoomsRange: dto.RoomsRange,
		CommercialBuildingLocation: dto.CommercialBuildingLocation,
		CommercialRentType: dto.CommercialRentType,
		Parameters: dto.Parameters,
	}
}

// Start реализует EventListenerPort, запуская прослушивание очереди.
func (a *ProcessedPropertyConsumerAdapter) Start(ctx context.Context) error {
	return a.consumer.StartConsuming(ctx)
}

// Close реализует EventListenerPort, корректно останавливая консьюмера.
func (a *ProcessedPropertyConsumerAdapter) Close() error {
	return a.consumer.Close()
}

// // unmarshalRecord - НОВАЯ функция-помощник для разбора ОДНОГО сообщения.
// // Мы вынесли эту логику из старого обработчика.
// func (a *ProcessedPropertyConsumerAdapter) unmarshalRecord(d amqp.Delivery) (*domain.RealEstateRecord, error) {
// 	var rawRecord tempRecord
// 	if err := json.Unmarshal(d.Body, &rawRecord); err != nil {
// 		return nil, fmt.Errorf("unmarshal tempRecord error: %w", err)
// 	}

// 	propertyRecord := &domain.RealEstateRecord{
// 		General: rawRecord.General,
// 	}

// 	// Ваша логика switch остается почти без изменений
// 	switch rawRecord.DetailsType {
// 	case "apartment":
// 		var details domain.Apartment
// 		if err := json.Unmarshal(rawRecord.Details, &details); err != nil {
// 			log.Printf("Error unmarshalling apartment details: %v", err)
// 			return nil, err
// 		}
// 		propertyRecord.Details = &details

// 	case "house":
// 		var houseDetails domain.House
// 		if err := json.Unmarshal(rawRecord.Details, &houseDetails); err != nil {
// 		    log.Printf("Error unmarshalling house details: %v", err)
// 		    return nil, err
// 		}
// 		propertyRecord.Details = &houseDetails

// 	case "commercial":
// 		var commercialDetails domain.Commercial
// 		if err := json.Unmarshal(rawRecord.Details, &commercialDetails); err != nil {
// 		    log.Printf("Error unmarshalling commercial details: %v", err)
// 		    return nil, err
// 		}
// 		propertyRecord.Details = &commercialDetails

// 	case "garage_and_parking":
// 		var garageParkingDetails domain.GarageAndParking
// 		if err := json.Unmarshal(rawRecord.Details, &garageParkingDetails); err != nil {
// 		    log.Printf("Error unmarshalling garage_and_parking details: %v", err)
// 		    return nil, err
// 		}
// 		propertyRecord.Details = &garageParkingDetails

// 	case "room":
// 		var roomDetails domain.Room
// 		if err := json.Unmarshal(rawRecord.Details, &roomDetails); err != nil {
// 		    log.Printf("Error unmarshalling room details: %v", err)
// 		    return nil, err
// 		}
// 		propertyRecord.Details = &roomDetails

// 	case "plot":
// 		var plotDetails domain.Plot
// 		if err := json.Unmarshal(rawRecord.Details, &plotDetails); err != nil {
// 		    log.Printf("Error unmarshalling plot details: %v", err)
// 		    return nil, err
// 		}
// 		propertyRecord.Details = &plotDetails

// 	case "new_building":
// 		var newBuidingDetails domain.NewBuilding
// 		if err := json.Unmarshal(rawRecord.Details, &newBuidingDetails); err != nil {
// 		    log.Printf("Error unmarshalling new_building details: %v", err)
// 		    return nil, err
// 		}
// 		propertyRecord.Details = &newBuidingDetails

// 	default:
// 		log.Printf("Unknown details type: %s", rawRecord.DetailsType)
// 		// Решите, что делать дальше: можно просто проигнорировать детали
// 		propertyRecord.Details = nil
// 	}

// 	return propertyRecord, nil
// }
