package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"kufar-parser-service/internal/contextkeys"
	"kufar-parser-service/internal/core/domain"
	"kufar-parser-service/internal/core/port"
	"real-estate-system/pkg/rabbitmq/rabbitmq_producer"

	// "log"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// TaskResultDTO - для сообщения в task_results_queue
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

func (a *TaskReporterAdapter) ReportResults(ctx context.Context, taskID uuid.UUID, stats *domain.ParsingTasksStats) error {

	logger := contextkeys.LoggerFromContext(ctx)
	// Обогащаем его информацией о компоненте
	adapterLogger := logger.WithFields(port.Fields{
		"component":   "TaskReporterAdapter",
		"routing_key": a.routingKey,
	})

	dto := TaskResultDTO{
		TaskID: taskID,
		Results: map[string]int{
			"searches_completed": stats.SearchesCompleted,
			"new_links_found":    stats.NewLinksFound,
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

	// Устанавливаем таймаут на операцию публикации, если контекст его не предоставляет
	publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Таймаут 10 секунд на публикацию
	defer cancel()

	// log.Printf("RabbitMQAdapter: Publishing report for task %s\n", taskID)
	adapterLogger.Info("Publishing report for task", nil)
	err := a.producer.Publish(publishCtx, a.routingKey, msg)
	if err != nil {
		adapterLogger.Error("Failed to publish report for task", err, nil)
		return fmt.Errorf("rabbitmq adapter: failed to publish report for task %s: %w", taskID, err)
	}

	adapterLogger.Info("Successfully published report for task", nil)
	return nil
}
