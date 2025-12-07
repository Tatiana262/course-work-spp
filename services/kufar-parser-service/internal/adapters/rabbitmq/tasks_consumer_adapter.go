package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"kufar-parser-service/internal/constants"
	"kufar-parser-service/internal/contextkeys"
	"kufar-parser-service/internal/core/domain"
	"kufar-parser-service/internal/core/port"
	usecases_port "kufar-parser-service/internal/core/port/usecases"

	// "sync"

	// "realt-parser-service/internal/core/usecase"
	"real-estate-system/pkg/rabbitmq/rabbitmq_common"
	"real-estate-system/pkg/rabbitmq/rabbitmq_consumer"

	// "log"

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
	// log.Printf("LinkConsumerAdapter: Received task (Tag: %d)\n", d.DeliveryTag)

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
}

func (a *TasksConsumerAdapter) translateDTOToInternalTasks(dto TaskInfo) ([]domain.SearchCriteria, error) {
	// 1. Находим СРЕЗ технических локаций для бизнес-региона
	kufarLocations, ok := constants.RegionToKufarMap[dto.Region]
	if !ok || len(kufarLocations) == 0 {
		return nil, fmt.Errorf("unknown or unconfigured region for Kufar: %s", dto.Region)
	}

	// 2. Находим технический ID категории
	kufarCategory, ok := constants.BusinessCategoryToKufarMap[dto.Category]
	if !ok {
		return nil, fmt.Errorf("unknown category for Kufar: %s", dto.Category)
	}

	// 3. Создаем задачи, итерируя по всем локациям И всем типам сделок
	tasks := make([]domain.SearchCriteria, 0, len(kufarLocations)*len(constants.DealTypes))

	// Внешний цикл по локациям (для "Минск" он выполнится дважды)
	for _, location := range kufarLocations {
		for _, dealType := range constants.DealTypes {

			if kufarCategory == constants.PlotCategory && dealType == constants.DealTypeRent ||
				kufarCategory == constants.NewBuildingCategory && dealType == constants.DealTypeRent {
				continue
			}

			task := domain.SearchCriteria{
				Category: kufarCategory,
				Location: location, // <-- Используем конкретную локацию из цикла
				DealType: dealType,

				AdsAmount: constants.MaxAdsAmount,
				SortBy:    constants.SortByDateDesc,

				Name: fmt.Sprintf("FindNew_%s_%s_loc-%s_%s", dto.Region, dto.Category, location, dealType),
			}
			tasks = append(tasks, task)
		}
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
