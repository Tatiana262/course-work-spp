package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"kufar-parser-service/internal/contextkeys"
	"kufar-parser-service/internal/core/domain"
	"kufar-parser-service/internal/core/port"
	"real-estate-system/pkg/rabbitmq/rabbitmq_producer"
	"reflect"

	// "log"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type DetailTranslator interface {
	Translate(details interface{}) (typeName string, detailsDTO interface{}, err error)
}

type ApartmentTranslator struct{}

func (t *ApartmentTranslator) Translate(details interface{}) (string, interface{}, error) {
	apt, ok := details.(*domain.Apartment)
	if !ok {
		return "", nil, fmt.Errorf("expected *domain.Apartment, got %T", details)
	}
	return "apartment", toApartmentDetailsDTO(apt), nil
}

type HouseTranslator struct{}

func (t *HouseTranslator) Translate(details interface{}) (string, interface{}, error) {
	house, ok := details.(*domain.House)
	if !ok {
		return "", nil, fmt.Errorf("expected *domain.House, got %T", details)
	}
	return "house", toHouseDetailsDTO(house), nil
}

type PassthroughTranslator struct {
	TypeName string
}

func (t *PassthroughTranslator) Translate(details interface{}) (string, interface{}, error) {
	// Здесь можно добавить проверку типа для безопасности, если нужно
	return t.TypeName, details, nil
}

// RabbitMQProcessedPropertyQueueAdapter для отправки обработанных объектов
type RabbitMQProcessedPropertyQueueAdapter struct {
	producer        *rabbitmq_producer.Publisher
	routingKey      string
	detailsRegistry map[reflect.Type]DetailTranslator
}

// NewRabbitMQProcessedPropertyQueueAdapter создает новый экземпляр
func NewRabbitMQProcessedPropertyQueueAdapter(producer *rabbitmq_producer.Publisher, routingKey string) (*RabbitMQProcessedPropertyQueueAdapter, error) {
	if producer == nil {
		return nil, fmt.Errorf("producer cannot be nil")
	}
	if routingKey == "" {
		return nil, fmt.Errorf("routingKey cannot be empty")
	}

	adapter := &RabbitMQProcessedPropertyQueueAdapter{
		producer:        producer,
		routingKey:      routingKey,
		detailsRegistry: make(map[reflect.Type]DetailTranslator),
	}

	adapter.detailsRegistry[reflect.TypeOf(&domain.Apartment{})] = &ApartmentTranslator{}
	adapter.detailsRegistry[reflect.TypeOf(&domain.House{})] = &HouseTranslator{}
	adapter.detailsRegistry[reflect.TypeOf(&domain.Commercial{})] = &PassthroughTranslator{TypeName: "commercial"}
	adapter.detailsRegistry[reflect.TypeOf(&domain.GarageAndParking{})] = &PassthroughTranslator{TypeName: "garage_and_parking"}
	adapter.detailsRegistry[reflect.TypeOf(&domain.Room{})] = &PassthroughTranslator{TypeName: "room"}
	adapter.detailsRegistry[reflect.TypeOf(&domain.Plot{})] = &PassthroughTranslator{TypeName: "plot"}
	adapter.detailsRegistry[reflect.TypeOf(&domain.NewBuilding{})] = &PassthroughTranslator{TypeName: "new_building"}

	return adapter, nil
}

// Enqueue отправляет PropertyRecord в очередь
func (a *RabbitMQProcessedPropertyQueueAdapter) Enqueue(ctx context.Context, record domain.RealEstateRecord, taskID uuid.UUID) error {

	logger := contextkeys.LoggerFromContext(ctx)
	adapterLogger := logger.WithFields(port.Fields{
		"component":   "RabbitMQProcessedPropertyQueueAdapter",
		"routing_key": a.routingKey,
		// "ad_id":       record.General.SourceAdID,
		// "task_id":     taskID,
	})

	// 1. Создаем DTO и маппим данные из домена в него.
	eventDTO := ProcessedRealEstateEventDTO{
		General: toGeneralDTO(record.General),
		// Details: record.Details,

		TaskID: taskID,
	}

	if record.Details != nil {
		// 1. Получаем тип деталей (например, reflect.TypeOf(&domain.Apartment{}))
		detailsType := reflect.TypeOf(record.Details)

		// 2. Ищем транслятор в нашем реестре
		translator, found := a.detailsRegistry[detailsType]
		if !found {
			err := fmt.Errorf("enqueue failed: unknown details type %T for source %s", record.Details, record.General.Source)
			adapterLogger.Error("Could not find translator for details type", err, port.Fields{"details_type": detailsType.String()})
			return err
		}

		// 3. Вызываем транслятор
		typeName, detailsDTO, err := translator.Translate(record.Details)
		if err != nil {
			adapterLogger.Error("Failed to translate details", err, port.Fields{"details_type": detailsType.String()})
			return err
		}

		eventDTO.DetailsType = typeName
		eventDTO.Details = detailsDTO
	}

	recordJSON, err := json.Marshal(eventDTO)
	if err != nil {
		adapterLogger.Error("Failed to marshal processed record to JSON", err, nil)
		return fmt.Errorf("failed to marshal processed record to JSON for URL %s: %w", record.General.AdLink, err)
	}

	msg := amqp.Publishing{
		ContentType:  "application/json",
		Body:         recordJSON,
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Headers: amqp.Table{
			"event-type":    "ProcessedRealEstateEvent", // Название события из схемы
			"event-version": "1.0.0",                    // Версия из схемы
		},
	}

	traceID := contextkeys.TraceIDFromContext(ctx)
	if traceID != "" {
		msg.Headers["x-trace-id"] = traceID
	}

	publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = a.producer.Publish(publishCtx, a.routingKey, msg)
	if err != nil {
		adapterLogger.Error("Failed to publish processed record", err, nil)
		return err
	}

	adapterLogger.Info("Successfully published processed record", port.Fields{"details_type": eventDTO.DetailsType})
	return nil
}

func toGeneralDTO(general domain.GeneralProperty) GeneralPropertyDTO {
	dto := GeneralPropertyDTO{ // Маппинг полей
		Source:     general.Source,
		SourceAdID: general.SourceAdID,
		// Category:   record.General.Category,
		AdLink:      general.AdLink,
		SaleType:    general.RemunerationType,
		Currency:    general.Currency,
		Images:      general.Images,
		ListTime:    general.ListTime,
		Description: general.Body,
		Title:       general.Subject,
		DealType:    general.DealType,

		Latitude:       general.Latitude,
		Longitude:      general.Longitude,
		CityOrDistrict: general.CityOrDistrict,
		Region:         general.Region,
		PriceBYN:       general.PriceBYN,
		PriceUSD:       general.PriceUSD,
		PriceEUR:       general.PriceEUR,
		Address:        general.Address,

		IsAgency:      general.IsAgency,
		SellerName:    general.SellerName,
		SellerDetails: general.SellerDetails,

		Status: general.Status,
	}

	return dto
}

func toApartmentDetailsDTO(d *domain.Apartment) ApartmentDetailsDTO {
	return ApartmentDetailsDTO{
		RoomsAmount:         d.RoomsAmount,
		FloorNumber:         d.FloorNumber,
		BuildingFloors:      d.BuildingFloors,
		TotalArea:           d.TotalArea,
		LivingSpaceArea:     d.LivingSpaceArea,
		KitchenArea:         d.KitchenArea,
		YearBuilt:           d.YearBuilt,
		WallMaterial:        d.WallMaterial,
		RepairState:         d.RepairState,
		BathroomType:        d.BathroomType,
		Balcony:             d.Balcony,
		PricePerSquareMeter: d.PricePerSquareMeter,
		Parameters:          d.Parameters,
	}
}

func toHouseDetailsDTO(d *domain.House) HouseDetailsDTO {
	return HouseDetailsDTO{
		TotalArea:         d.TotalArea,
		PlotArea:          d.PlotArea,
		WallMaterial:      d.WallMaterial,
		YearBuilt:         d.YearBuilt,
		LivingSpaceArea:   d.LivingSpaceArea,
		BuildingFloors:    d.BuildingFloors,
		RoomsAmount:       d.RoomsAmount,
		KitchenArea:       d.KitchenArea,
		Electricity:       d.Electricity,
		Water:             d.Water,
		Heating:           d.Heating,
		Sewage:            d.Sewage,
		Gaz:               d.Gaz,
		RoofMaterial:      d.RoofMaterial,
		HouseType:         d.HouseType,
		CompletionPercent: d.CompletionPercent,
		Parameters:        d.Parameters,
	}
}
