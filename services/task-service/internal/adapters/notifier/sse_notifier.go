package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"task-service/internal/contextkeys"
	"task-service/internal/core/port"
)

// clientChannel - это канал, через который мы будем отправлять события одному конкретному клиенту (браузеру)
type clientChannel chan []byte

// структура для передачи в канал
type eventWithContext struct {
	ctx   context.Context
	event port.TaskEvent
}

// SSENotifier - это реализация NotifierPort
type SSENotifier struct {
	// clients хранит активные подключения. Ключ - ID пользователя,
	// значение - срез каналов (один пользователь может открыть несколько вкладок)
	clients map[string][]clientChannel
	// mu - мьютекс для защиты clients от одновременного доступа из разных горутин
	mu sync.RWMutex

	// eventChan - внутренний канал, в который Use Cases будут бросать события
	eventChan chan eventWithContext

	logger    port.LoggerPort
}


// NewSSENotifier создает и запускает новый нотификатор
func NewSSENotifier(baseLogger port.LoggerPort) *SSENotifier {

	notifierLogger := baseLogger.WithFields(port.Fields{"component": "SSENotifier"})

	notifier := &SSENotifier{
		clients:   make(map[string][]clientChannel),
		eventChan: make(chan eventWithContext, 100), // Буферизованный канал
		logger:    notifierLogger,
	}

	// Запускаем основную горутину-диспетчер, которая будет слушать события и рассылать их
	go notifier.dispatcher()

	return notifier
}

// dispatcher - работает в фоне и никогда не завершается
func (n *SSENotifier) dispatcher() {
	n.logger.Debug("Notifier dispatcher started.", nil)
	for {
		
		// Блокируемся, пока не придет новое событие из Use Case
		eventPackage := <-n.eventChan

		ctx := eventPackage.ctx
		event := eventPackage.event

		// Извлекаем логгер из переданного контекста
		loggerFromCtx := contextkeys.LoggerFromContext(ctx)

		// Создаем логгер для этого события, обогащая его данными из события
		eventLogger := loggerFromCtx.WithFields(port.Fields{
			"component":  "SSENotifier.dispatcher",
			"event_type": event.Type,
			"task_id":    event.Data.ID.String(),
		})
		
		eventLogger.Info("Processing new event.", nil)

		// Маршалим событие в JSON
		eventBytes, err := json.Marshal(event.Data)
		if err != nil {
			eventLogger.Error("Failed to marshal event", err, nil)
			continue
		}
		
		// Форматируем для SSE
		sseMessage := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(eventBytes)))

		// Получаем ID пользователя, которому адресовано событие
		userID := event.Data.CreatedByUserID.String()

		// Блокируем clients для безопасного чтения
		n.mu.RLock()
		
		// Находим все активные соединения для этого пользователя
		if clientChannels, found := n.clients[userID]; found {
			eventLogger.Debug("Dispatching event to clients", port.Fields{"user_id": userID, "channels_count": len(clientChannels)})
			// Отправляем сообщение в каждый канал (в каждую открытую вкладку)
			for _, ch := range clientChannels {
				// Используем select с default, чтобы не заблокироваться,
				// если канал клиента переполнен или закрыт
				select {
				case ch <- sseMessage:
				default:
					eventLogger.Warn("Client channel is full or closed, skipping.", port.Fields{"user_id": userID})
				}
			}
		} else {
			eventLogger.Debug("No active clients for user, event dropped.", port.Fields{"user_id": userID})
		}
		
		n.mu.RUnlock()
	}
}

// Notify - это реализация метода из NotifierPort
// Use Cases вызывают этот метод. Он просто отправляет событие во внутренний канал
func (n *SSENotifier) Notify(ctx context.Context, event port.TaskEvent) {
	eventPackage := eventWithContext{
		ctx:   ctx,
		event: event,
	}

	// Отправка в канал неблокирующая, если есть место в буфере
	n.eventChan <- eventPackage
}

// AddClient добавляет нового клиента (новое SSE-соединение)
// Этот метод вызывается из HTTP-хендлера
func (n *SSENotifier) AddClient(userID string) clientChannel {
	n.mu.Lock()
	defer n.mu.Unlock()

	ch := make(clientChannel, 100) // Канал для одного клиента
	n.clients[userID] = append(n.clients[userID], ch)

	n.logger.Info("Client connected for user", port.Fields{
		"user_id":         userID,
		"total_connections_for_user": len(n.clients[userID]),
	})
	
	return ch
}

// RemoveClient удаляет канал клиента при отключении
// Этот метод будет вызывается из HTTP-хендлера, когда клиент закрывает соединение
func (n *SSENotifier) RemoveClient(userID string, ch clientChannel) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if channels, found := n.clients[userID]; found {
		newChannels := make([]clientChannel, 0)
		for _, c := range channels {
			// Сравниваем указатели на каналы, чтобы найти и удалить нужный
			if c != ch {
				newChannels = append(newChannels, c)
			}
		}

		if len(newChannels) == 0 {
			delete(n.clients, userID)
			n.logger.Debug("Last client disconnected for user. User removed.", port.Fields{"user_id": userID})
		} else {
			n.clients[userID] = newChannels
			n.logger.Info("Client disconnected for user.", port.Fields{
				"user_id":             userID,
				"remaining_connections": len(newChannels),
			})
		}
	}
}