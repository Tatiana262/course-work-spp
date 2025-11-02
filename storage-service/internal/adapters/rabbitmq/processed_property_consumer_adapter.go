package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"storage-service/internal/contracts"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/usecase"
	"storage-service/pkg/rabbitmq/rabbitmq_consumer"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)


// DTO - структура, которая соответствует JSON-схеме
type IncomingEventDTO struct {
	General     GeneralPropertyDTO `json:"general"`
	DetailsType string             `json:"details_type"`
	Details     json.RawMessage    `json:"details"`
}

type GeneralPropertyDTO struct {
	Source     string `json:"source"`
	SourceAdID int64  `json:"sourceAdId"`
	Category         string    `json:"category"`
	AdLink           string    `json:"adLink"`
	SaleType 		string    `json:"saleType"`
	Currency         string    `json:"currency"`
	Images           []string  `json:"images"`
	ListTime         time.Time `json:"listTime"`
	Description             string    `json:"description"`
	Title          string    `json:"title"`
	DealType         string    `json:"dealType"`
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	CityOrDistrict   string    `json:"cityOrDistrict"`
	Region           string    `json:"region"`
	PriceBYN         float64   `json:"priceBYN"`
	PriceUSD         float64   `json:"priceUSD"`
	PriceEUR         *float64  `json:"priceEUR,omitempty"`
	Address        string  `json:"address"`


	IsAgency        bool      `json:"isAgency"`
	SellerName     string  	  `json:"sellerName"`
	SellerDetails	json.RawMessage `json:"sellerDetails"`
}

// ProcessedPropertyConsumerAdapter - это входящий адаптер, который слушает очередь 
// с обработанными объектами недвижимости и вызывает use case для их сохранения
type ProcessedPropertyConsumerAdapter struct {
	consumer rabbitmq_consumer.Consumer
	useCase  *usecase.SavePropertyUseCase
}

// NewProcessedPropertyConsumerAdapter создает новый адаптер
func NewProcessedPropertyConsumerAdapter(
	consumerCfg rabbitmq_consumer.ConsumerConfig,
	useCase *usecase.SavePropertyUseCase,
) (*ProcessedPropertyConsumerAdapter, error) {

	adapter := &ProcessedPropertyConsumerAdapter{
		useCase: useCase,
	}

	// Создаем consumer, передавая ему метод этого адаптера как обработчик
	consumer, err := rabbitmq_consumer.NewBatchConsumer(consumerCfg, adapter.batchMessageHandler, 100, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ consumer for processed properties: %w", err)
	}
	adapter.consumer = consumer

	return adapter, nil
}

// batchMessageHandler - обработчик, который принимает срез сообщений
func (a *ProcessedPropertyConsumerAdapter) batchMessageHandler(deliveries []amqp.Delivery) error {
	records := make([]domain.RealEstateRecord, 0, len(deliveries))
	log.Printf("ProcessedPropertyBatchConsumerAdapter: Received batch of %d messages to process.\n", len(deliveries))

	// Разбираем все сообщения в пачке
	for _, d := range deliveries {
		record, err := a.unmarshalRecord(d)
		if err != nil {
			log.Printf("FATAL: Failed to unmarshal message in batch, tag %d: %v. This will cause requeue of the whole batch.", d.DeliveryTag, err)
			return err
		}
		if record != nil {
			records = append(records, *record)
		}
	}

	if len(records) == 0 {
		log.Println("ProcessedPropertyBatchConsumerAdapter: No valid records in batch to save.")
		return nil
	}

	log.Printf("ProcessedPropertyBatchConsumerAdapter: Calling BatchSave for %d records...", len(records))
	return a.useCase.BatchSave(context.Background(), records)
}

// unmarshalRecord -  функция для разбора одного сообщения
func (a *ProcessedPropertyConsumerAdapter) unmarshalRecord(d amqp.Delivery) (*domain.RealEstateRecord, error) {
	// Валидация по схеме
	eventType, _ := d.Headers["event-type"].(string)
	eventVersion, _ := d.Headers["event-version"].(string)
	if err := contracts.ValidateEvent(eventType, eventVersion, d.Body); err != nil {
		log.Printf("FATAL: Message failed schema validation: %v. Rejecting.", err)
		return nil, err
	}

	// Десериализация в DTO
	var dto IncomingEventDTO
	if err := json.Unmarshal(d.Body, &dto); err != nil {
		return nil, fmt.Errorf("failed to unmarshal incoming event DTO: %w", err)
	}

	// Трансляция из DTO в домен
	record := &domain.RealEstateRecord{
		General: toDBGeneralProperty(dto),
	}

	switch dto.DetailsType {
	case "apartment":
		var details domain.Apartment
		if err := json.Unmarshal(dto.Details, &details); err != nil {
			log.Printf("Error unmarshalling apartment details: %v", err)
			return nil, err
		}
		record.Details = &details

	case "house":
		var houseDetails domain.House
		if err := json.Unmarshal(dto.Details, &houseDetails); err != nil {
			log.Printf("Error unmarshalling house details: %v", err)
			return nil, err
		}
		record.Details = &houseDetails

	case "commercial":
		var commercialDetails domain.Commercial
		if err := json.Unmarshal(dto.Details, &commercialDetails); err != nil {
			log.Printf("Error unmarshalling commercial details: %v", err)
			return nil, err
		}
		record.Details = &commercialDetails

	case "garage_and_parking":
		var garageParkingDetails domain.GarageAndParking
		if err := json.Unmarshal(dto.Details, &garageParkingDetails); err != nil {
			log.Printf("Error unmarshalling garage_and_parking details: %v", err)
			return nil, err
		}
		record.Details = &garageParkingDetails

	case "room":
		var roomDetails domain.Room
		if err := json.Unmarshal(dto.Details, &roomDetails); err != nil {
			log.Printf("Error unmarshalling room details: %v", err)
			return nil, err
		}
		record.Details = &roomDetails

	case "plot":
		var plotDetails domain.Plot
		if err := json.Unmarshal(dto.Details, &plotDetails); err != nil {
			log.Printf("Error unmarshalling plot details: %v", err)
			return nil, err
		}
		record.Details = &plotDetails

	case "new_building":
		var newBuidingDetails domain.NewBuilding
		if err := json.Unmarshal(dto.Details, &newBuidingDetails); err != nil {
			log.Printf("Error unmarshalling new_building details: %v", err)
			return nil, err
		}
		record.Details = &newBuidingDetails

	default:
		log.Printf("Unknown details type: %s", dto.DetailsType)
		record.Details = nil
	}

	return record, nil
}

func toDBGeneralProperty(dto IncomingEventDTO) domain.GeneralProperty {

	wktPoint := fmt.Sprintf("SRID=4326;POINT(%f %f)", dto.General.Longitude, dto.General.Latitude)

	return domain.GeneralProperty{ 
		ID:               uuid.New(),
		Source:           dto.General.Source,
		SourceAdID:       dto.General.SourceAdID,
		UpdatedAt:        time.Now(),
		CreatedAt:        time.Now(),
		Category:         dto.General.Category,
		AdLink:           dto.General.AdLink,
		SaleType:         dto.General.SaleType,
		Currency:         dto.General.Currency,
		Images:           dto.General.Images,
		ListTime:         dto.General.ListTime,
		Description:      dto.General.Description,
		Title:            dto.General.Title,
		DealType:         dto.General.DealType,
		Coordinates:      wktPoint,
		CityOrDistrict:   dto.General.CityOrDistrict,
		Region:           dto.General.Region,
		PriceBYN:         dto.General.PriceBYN,
		PriceUSD:         dto.General.PriceUSD,
		PriceEUR:         dto.General.PriceEUR,
		Address:          dto.General.Address,

		IsAgency:        dto.General.IsAgency,
		SellerName:       dto.General.SellerName,
		SellerDetails:    dto.General.SellerDetails,

		Latitude:         dto.General.Latitude,
		Longitude:        dto.General.Longitude,
	}

}

// Start реализует EventListenerPort, запуская прослушивание очереди
func (a *ProcessedPropertyConsumerAdapter) Start(ctx context.Context) error {
	return a.consumer.StartConsuming(ctx)
}

// Close реализует EventListenerPort, корректно останавливая консьюмера
func (a *ProcessedPropertyConsumerAdapter) Close() error {
	return a.consumer.Close()
}