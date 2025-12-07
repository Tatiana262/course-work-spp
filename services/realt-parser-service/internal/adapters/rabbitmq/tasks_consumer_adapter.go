package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"realt-parser-service/internal/constants"
	"realt-parser-service/internal/contextkeys"
	"realt-parser-service/internal/core/domain"
	"realt-parser-service/internal/core/port"
	usecases_port "realt-parser-service/internal/core/port/usecases"

	// "realt-parser-service/internal/core/usecase"
	"real-estate-system/pkg/rabbitmq/rabbitmq_common"
	"real-estate-system/pkg/rabbitmq/rabbitmq_consumer"

	// "strings"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// LinkConsumerAdapter - это входящий адаптер, который слушает очередь
// со ссылками и вызывает use case для их обработки.
type TasksConsumerAdapter struct {
	consumer      rabbitmq_consumer.Consumer
	orchestrateUC usecases_port.OrchestrateParsingPort
	logger        port.LoggerPort
}

// NewLinkConsumerAdapter создает новый адаптер.
func NewTasksConsumerAdapter(
	consumerCfg rabbitmq_consumer.ConsumerConfig,
	orchestrateUC usecases_port.OrchestrateParsingPort,
	logger port.LoggerPort,
	connManager *rabbitmq_common.ConnectionManager,
) (*TasksConsumerAdapter, error) {

	adapter := &TasksConsumerAdapter{
		orchestrateUC: orchestrateUC,
		logger:        logger,
	}

	// 1. Создаем логгер для pkg-уровня с контекстом нашего компонента
	pkgLogger := logger.WithFields(port.Fields{"component": "rabbitmq_distributing_consumer", "consumer_tag": consumerCfg.ConsumerTag})
	// 2. Создаем мост и передаем его в конфиг
	consumerCfg.Logger = NewPkgLoggerBridge(pkgLogger)

	consumer, err := rabbitmq_consumer.NewDistributingConsumer(consumerCfg, adapter.messageHandler, connManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ consumer for links: %w", err)
	}
	adapter.consumer = consumer

	return adapter, nil
}

// messageHandler - приватный метод адаптера.
func (a *TasksConsumerAdapter) messageHandler(d amqp.Delivery) (err error) {
	traceID, ok := d.Headers["x-trace-id"].(string)
	if !ok || traceID == "" {
		traceID = uuid.New().String()
	}

	// Создаем логгер для этого конкретного сообщения
	msgLogger := a.logger.WithFields(port.Fields{
		"trace_id":     traceID,
		"delivery_tag": d.DeliveryTag,
	})

	// Создаем контекст и кладем в него логгер
	ctx := context.Background()
	ctx = contextkeys.ContextWithLogger(ctx, msgLogger)
	ctx = contextkeys.ContextWithTraceID(ctx, traceID)

	msgLogger.Info("Received new task", nil)

	var taskDTO TaskInfo
	if err := json.Unmarshal(d.Body, &taskDTO); err != nil {
		msgLogger.Error("Error unmarshalling task DTO, NACKing message", err, nil)
		// Ошибка разбора JSON - это ПОСТОЯННАЯ ошибка. Нет смысла повторять.
		return fmt.Errorf("unmarshal error: %w", err)
	}

	// Обогащаем логгер ID задачи для всех последующих сообщений
	taskLogger := msgLogger.WithFields(port.Fields{"task_id": taskDTO.TaskID.String()})
	ctx = contextkeys.ContextWithLogger(ctx, taskLogger) // Обновляем контекст с еще более детальным логгером

	// 1. Получаем СРЕЗ задач от транслятора
	internalTasks, err := a.translateDTOToInternalTasks(taskDTO)
	if err != nil {
		taskLogger.Error("Cannot translate DTO to internal tasks", err, nil)
		return nil
	}

	if err := a.orchestrateUC.Execute(ctx, internalTasks, taskDTO.TaskID); err != nil {
		msgLogger.Error("Orchestration use case failed", err, nil)
		return err // Возвращаем ошибку для retry
	}

	return nil

	// for _, task := range internalTasks {
	//     // Создаем отдельный контекст для каждой задачи, если нужно
	// 	taskCtx := context.Background()
	// 	log.Printf("Executing task: %s", task.Name)
	// 	if err := a.useCase.Execute(taskCtx, task, taskDTO.TaskID); err != nil {
	// 		log.Printf("ERROR: Task '%s' failed: %v", task.Name, err)
	// 	}
	// }

	// log.Printf("All %d tasks for search new objects completed.", len(internalTasks))

	// return nil
}

type Price struct {
	From     int `json:"from"`
	To       int `json:"to"`
	Currency int `json:"currency"`
}

func (a *TasksConsumerAdapter) translateDTOToInternalTasks(dto TaskInfo) ([]domain.SearchCriteria, error) {

	// 2. Находим технический ID категории
	locationUUID, ok := constants.RegionToRealtMap[dto.Region]
	if !ok {
		return nil, fmt.Errorf("unknown region for Realt: %s", dto.Region)
	}

	// 1. Находим шаблоны для запрошенной бизнес-категории
	templates, ok := constants.BusinessCategoryToTemplatesMap[dto.Category]
	if !ok {
		return nil, fmt.Errorf("no search templates found for category: %s", dto.Category)
	}

	// 2. Создаем итоговый срез задач
	tasks := make([]domain.SearchCriteria, 0, len(templates))

	// 3. Для каждого шаблона создаем полноценную задачу
	for _, tmpl := range templates {
		task := domain.SearchCriteria{
			// Общие параметры из DTO
			LocationUUID: locationUUID,
			Page:         1,
			// Уникальные параметры из шаблона
			Category:       tmpl.Category,
			ObjectCategory: tmpl.ObjectCategory, // Будет nil, если в шаблоне не задано
			ObjectType:     tmpl.ObjectType,     // Будет nil, если в шаблоне не задано

			// FOR DEBUG
			// для квартир
			Rooms: []int{5},
			// для домов
			// Price: Price{
			// 	From: 1000,
			// 	To: 20000,
			// 	Currency: 840,
			// },
			// Генерируем имя для логов
			Name: fmt.Sprintf("FindNew_%s_%s", dto.Region, tmpl.Name),
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// Start реализует EventListenerPort
func (a *TasksConsumerAdapter) Start(ctx context.Context) error {
	return a.consumer.StartConsuming(ctx)
}

// Close реализует EventListenerPort
func (a *TasksConsumerAdapter) Close() error {
	return a.consumer.Close()
}
