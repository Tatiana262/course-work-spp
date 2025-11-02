package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"kufar-parser-service/internal/core/domain"
	"kufar-parser-service/pkg/rabbitmq/rabbitmq_producer"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// DTO точно соответствует JSON-схеме processed-real-estate/v1.json
type ProcessedRealEstateEventDTO struct {
    General     GeneralPropertyDTO `json:"general"`
    DetailsType string             `json:"details_type"`
    Details     interface{}        `json:"details"`
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
	SellerDetails	map[string]interface{} `json:"sellerDetails"`
}

// RabbitMQProcessedPropertyQueueAdapter для отправки обработанных объектов
type RabbitMQProcessedPropertyQueueAdapter struct {
	producer   *rabbitmq_producer.Publisher
	routingKey string
}

// NewRabbitMQProcessedPropertyQueueAdapter создает новый экземпляр
func NewRabbitMQProcessedPropertyQueueAdapter(producer *rabbitmq_producer.Publisher, routingKey string) (*RabbitMQProcessedPropertyQueueAdapter, error) {
	if producer == nil { return nil, fmt.Errorf("producer cannot be nil") }
	if routingKey == "" { return nil, fmt.Errorf("routingKey cannot be empty") }
	return &RabbitMQProcessedPropertyQueueAdapter{
		producer:   producer,
		routingKey: routingKey,
	}, nil
}

// Enqueue отправляет RealEstateRecord в очередь
func (a *RabbitMQProcessedPropertyQueueAdapter) Enqueue(ctx context.Context, record domain.RealEstateRecord) error {

	 // Создаем DTO и маппим данные из домена в него
	 eventDTO := ProcessedRealEstateEventDTO{
        General: GeneralPropertyDTO{ 
            Source:     record.General.Source,
            SourceAdID: record.General.SourceAdID,
			Category:   record.General.Category,
            AdLink:     record.General.AdLink,
			SaleType: record.General.RemunerationType,
			Currency: 	record.General.Currency,
			Images:     record.General.Images,
			ListTime:   record.General.ListTime, 
			Description:		record.General.Body,
            Title:    record.General.Subject,
			DealType:   record.General.DealType,
			
			Latitude: record.General.Latitude,
			Longitude: record.General.Longitude,
			CityOrDistrict: record.General.CityOrDistrict,
			Region: record.General.Region,
			PriceBYN: record.General.PriceBYN,
            PriceUSD:   record.General.PriceUSD,
			PriceEUR: record.General.PriceEUR,
			Address: record.General.Address,
            
			IsAgency: record.General.IsAgency,
            SellerName: record.General.SellerName,
			SellerDetails: record.General.SellerDetails,
		
        },
        Details: record.Details,
    }

	// Определяем тип и заполняем поле DetailsType
	switch record.Details.(type) {
	case *domain.Apartment:
		eventDTO.DetailsType = "apartment"
	case *domain.House:
		eventDTO.DetailsType = "house"
	case *domain.Commercial:
		eventDTO.DetailsType = "commercial"
	case *domain.GarageAndParking:
		eventDTO.DetailsType = "garage_and_parking"
	case *domain.Room:
		eventDTO.DetailsType = "room"
	case *domain.Plot:
		eventDTO.DetailsType = "plot"
	case *domain.NewBuilding:
		eventDTO.DetailsType = "new_building"
	default:
        if record.Details != nil {
             return fmt.Errorf("enqueue failed: unknown details type %T for source %s", record.Details, record.General.Source)
        }
	}

	recordJSON, err := json.Marshal(eventDTO)
	if err != nil {
		return fmt.Errorf("failed to marshal property record to JSON for URL %s: %w", record.General.Source, err)
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

	publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	log.Printf("ProcessedPropertyQueue: Publishing processed record for URL '%s' to key '%s'\n", record.General.Source, a.routingKey)
	return a.producer.Publish(publishCtx, a.routingKey, msg)
}