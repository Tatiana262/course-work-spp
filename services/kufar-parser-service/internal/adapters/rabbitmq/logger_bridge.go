package rabbitmq

import (
	"kufar-parser-service/internal/core/port"
	"real-estate-system/pkg/rabbitmq/rabbitmq_common"
)

// PkgLoggerBridge адаптирует наш внутренний LoggerPort к интерфейсу pkg-уровня.
type PkgLoggerBridge struct {
	internalLogger port.LoggerPort
}

// NewPkgLoggerBridge создает новый мост.
func NewPkgLoggerBridge(logger port.LoggerPort) rabbitmq_common.Logger {
	return &PkgLoggerBridge{internalLogger: logger}
}

func (b *PkgLoggerBridge) toFields(keysAndValues ...interface{}) port.Fields {
	fields := make(port.Fields, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok || i+1 >= len(keysAndValues) {
			continue // Пропускаем некорректные пары
		}
		fields[key] = keysAndValues[i+1]
	}
	return fields
}

func (b *PkgLoggerBridge) Debug(msg string, keysAndValues ...interface{}) {
	// В нашем порте нет Debug, поэтому отправляем как Info
	b.internalLogger.Debug(msg, b.toFields(keysAndValues...))
}

func (b *PkgLoggerBridge) Info(msg string, keysAndValues ...interface{}) {
	b.internalLogger.Info(msg, b.toFields(keysAndValues...))
}

func (b *PkgLoggerBridge) Warn(msg string, keysAndValues ...interface{}) {
	b.internalLogger.Warn(msg, b.toFields(keysAndValues...))
}

func (b *PkgLoggerBridge) Error(err error, msg string, keysAndValues ...interface{}) {
	b.internalLogger.Error(msg, err, b.toFields(keysAndValues...))
}
