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
	"real-estate-system/pkg/rabbitmq/rabbitmq_common"
	"real-estate-system/pkg/rabbitmq/rabbitmq_consumer"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)


type TasksConsumerAdapter struct {
	consumer      rabbitmq_consumer.Consumer
	orchestrateUC usecases_port.OrchestrateParsingPort
	logger        port.LoggerPort
}


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

	// Создаем логгер для pkg-уровня с контекстом нашего компонента
	pkgLogger := logger.WithFields(port.Fields{"component": "rabbitmq_distributing_consumer", "consumer_tag": consumerCfg.ConsumerTag})
	// Создаем мост и передаем его в конфиг
	consumerCfg.Logger = NewPkgLoggerBridge(pkgLogger)

	consumer, err := rabbitmq_consumer.NewDistributingConsumer(consumerCfg, adapter.messageHandler, connManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ consumer for links: %w", err)
	}
	adapter.consumer = consumer

	return adapter, nil
}

// messageHandler - приватный метод адаптера
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

	msgLogger.Info("Received new task for find objects", nil)

	var taskDTO TaskInfo
	if err := json.Unmarshal(d.Body, &taskDTO); err != nil {
		msgLogger.Error("Error unmarshalling task DTO, NACKing message", err, nil)
		// Ошибка разбора JSON - это постоянная ошибка, нет смысла повторять обработку сообщения
		return fmt.Errorf("unmarshal error: %w", err)
	}

	taskLogger := msgLogger.WithFields(port.Fields{"task_id": taskDTO.TaskID.String()})
	ctx = contextkeys.ContextWithLogger(ctx, taskLogger) 

	// Получаем срез задач от транслятора
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
	// Находим срез технических локаций для бизнес-региона
	kufarLocations, ok := constants.RegionToKufarMap[dto.Region]
	if !ok || len(kufarLocations) == 0 {
		return nil, fmt.Errorf("unknown or unconfigured region for Kufar: %s", dto.Region)
	}

	// Находим срез технических ID для категории
	kufarCategories, ok := constants.BusinessCategoryToKufarMap[dto.Category]
	if !ok {
		return nil, fmt.Errorf("unknown category for Kufar: %s", dto.Category)
	}

	// Создаем задачи
	tasks := make([]domain.SearchCriteria, 0, len(kufarLocations)*len(constants.DealTypes))

	for _, location := range kufarLocations {
		for _, kufarCategory := range kufarCategories {
			for _, dealType := range constants.DealTypes {

				if kufarCategory == constants.PlotCategory && dealType == constants.DealTypeRent ||
					kufarCategory == constants.NewBuildingCategory && dealType == constants.DealTypeRent ||
					kufarCategory == constants.TravelsCategory && dealType == constants.DealTypeRent{
					continue
				}
	
				task := domain.SearchCriteria{
					Category: kufarCategory,
					Location: location,
					DealType: dealType,
	
					AdsAmount: constants.MaxAdsAmount,
					SortBy:    constants.SortByDateDesc,
	
					Name: fmt.Sprintf("FindNew_%s_%s_loc-%s_%s", dto.Region, dto.Category, location, dealType),
				}
	
				if kufarCategory == constants.TravelsCategory {
					query := constants.Queries[dto.Category]
					task.Query = query
				}
				
				tasks = append(tasks, task)
			}
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
