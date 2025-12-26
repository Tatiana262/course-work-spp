package rabbitmq_common

import (
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ConnectionManager управляет единственным соединением RabbitMQ
type ConnectionManager struct {
	url        string
	connection *amqp.Connection
	mutex      sync.RWMutex
	Logger     Logger
}

var (
	managerInstance *ConnectionManager
	once            sync.Once
)

// GetManager создает или возвращает глобальный экземпляр менеджера (Синглтон)
func GetManager(url string, logger Logger) (*ConnectionManager, error) {
	var initErr error

	once.Do(func() {
		if logger == nil {
			logger = NewNoopLogger()
		}
		managerInstance = &ConnectionManager{
			url:    url,
			Logger: logger,
		}
		// Пытаемся подключиться при инициализации
		if _, err := managerInstance.getConnection(); err != nil {
			logger.Error(err, "Initial connection failed")
			initErr = fmt.Errorf("initial connection failed: %w", err)
			return
		}
		// Запускаем в фоне мониторинг и переподключение
		go managerInstance.handleReconnect()
	})

	if initErr != nil {
		return nil, initErr
	}

	return managerInstance, nil
}

// getConnection возвращает существующее соединение или пытается его установить
func (m *ConnectionManager) getConnection() (*amqp.Connection, error) {
	m.mutex.RLock()
	if m.connection != nil && !m.connection.IsClosed() {
		m.mutex.RUnlock()
		return m.connection, nil
	}
	m.mutex.RUnlock()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Повторная проверка, вдруг другой поток уже успел переподключиться
	if m.connection != nil && !m.connection.IsClosed() {
		return m.connection, nil
	}

	m.Logger.Debug("ConnectionManager: Connecting...")
	conn, err := amqp.Dial(m.url)
	if err != nil {
		return nil, fmt.Errorf("ConnectionManager: failed to dial RabbitMQ: %w", err)
	}
	m.connection = conn
	m.Logger.Debug("ConnectionManager: Connected successfully!")
	return m.connection, nil
}

// GetChannel - основной метод для получения нового канала из общего соединения
func (m *ConnectionManager) GetChannel() (*amqp.Connection, *amqp.Channel, error) {
	conn, err := m.getConnection()
	if err != nil {
		return nil, nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return conn, nil, fmt.Errorf("ConnectionManager: failed to open a channel: %w", err)
	}
	return conn, ch, nil
}

func (m *ConnectionManager) handleReconnect() {
	for {
		// Ждем секунд перед проверкой
		time.Sleep(10 * time.Second)

		m.mutex.RLock()
		// Если соединения нет или оно не закрыто, ничего не делаем
		if m.connection == nil || !m.connection.IsClosed() {
			m.mutex.RUnlock()
			continue
		}
		m.mutex.RUnlock()

		m.Logger.Debug("ConnectionManager: Detected closed connection. Attempting to reconnect...")
		// Пытаемся переподключиться
		if _, err := m.getConnection(); err != nil {
			m.Logger.Error(err, "ConnectionManager: Reconnect failed")
		}
	}
}

// Close закрывает общее соединение RabbitMQ
func (m *ConnectionManager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.connection != nil && !m.connection.IsClosed() {
		m.Logger.Debug("ConnectionManager: Closing the connection...")
		err := m.connection.Close()
		if err != nil {
			m.Logger.Error(err, "ConnectionManager: Failed to close connection properly")
			return err
		}
		m.Logger.Debug("ConnectionManager: Connection closed successfully.")
		return nil
	}

	m.Logger.Debug("ConnectionManager: Connection was already closed or not established.")
	return nil
}
