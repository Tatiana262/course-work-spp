package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"real-estate-system/pkg/rabbitmq/rabbitmq_producer"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)


type TaskResultDTO struct {
	TaskID  uuid.UUID      `json:"task_id"`
	Results map[string]int `json:"results"`
}

type TaskReporterAdapter struct {
	producer   *rabbitmq_producer.Publisher
	routingKey string
}

func NewTaskReporterAdapter(producer *rabbitmq_producer.Publisher, routingKey string) (*TaskReporterAdapter, error) {
	if producer == nil {
		return nil, fmt.Errorf("rabbitmq adapter: producer cannot be nil")
	}
	if routingKey == "" {
		return nil, fmt.Errorf("rabbitmq adapter: routingKey cannot be empty")
	}
	return &TaskReporterAdapter{
		producer:   producer,
		routingKey: routingKey,
	}, nil
}

func (a *TaskReporterAdapter) ReportResults(ctx context.Context, taskID uuid.UUID, stats *domain.BatchSaveStats) error {
	// Извлекаем и обогащаем логгер
	logger := contextkeys.LoggerFromContext(ctx)
	adapterLogger := logger.WithFields(port.Fields{
		"component":   "TaskReporterAdapter",
		"routing_key": a.routingKey,
		"task_id":     taskID.String(),
	})

	dto := TaskResultDTO{
		TaskID: taskID,
		Results: map[string]int{
			"created":         stats.Created,
			"updated":         stats.Updated,
			"archived":        stats.Archived,
			"total_processed": stats.Created + stats.Updated + stats.Archived,
		},
	}

	body, _ := json.Marshal(dto)

	msg := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent, // Для сохранения сообщений при перезапуске брокера
		Timestamp:    time.Now(),
		Headers:      make(amqp.Table),
	}

	traceID := contextkeys.TraceIDFromContext(ctx)
	if traceID != "" {
		msg.Headers["x-trace-id"] = traceID
	}

	publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Таймаут 10 секунд на публикацию
	defer cancel()

	adapterLogger.Debug("Publishing batch save report for task", port.Fields{"stats": dto.Results})
	err := a.producer.Publish(publishCtx, a.routingKey, msg)
	if err != nil {
		adapterLogger.Error("Failed to publish report", err, nil)
		return fmt.Errorf("rabbitmq adapter: failed to publish report for task %s: %w", taskID, err)
	}

	adapterLogger.Info("Successfully published report", port.Fields{"stats": dto.Results})
	return nil
}
