package logger_adapter

import (
	"actualization-service/internal/core/port"
	"fmt"
)

type MultiLoggerAdapter struct {
	loggers []port.LoggerPort
}

func NewMultiloggerAdapter(loggers ...port.LoggerPort) (port.LoggerPort, error) {
	if len(loggers) == 0 {
		return nil, fmt.Errorf("multilogger: at least one logger is required")
	}
	return &MultiLoggerAdapter{loggers: loggers}, nil
}

func (m *MultiLoggerAdapter) Info(msg string, fields port.Fields) {
	for _, logger := range m.loggers {
		logger.Info(msg, fields)
	}
}

func (m *MultiLoggerAdapter) Warn(msg string, fields port.Fields) {
	for _, logger := range m.loggers {
		logger.Warn(msg, fields)
	}
}

func (m *MultiLoggerAdapter) Error(msg string, err error, fields port.Fields) {
	for _, logger := range m.loggers {
		logger.Error(msg, err, fields)
	}
}

func (m *MultiLoggerAdapter) Debug(msg string, fields port.Fields) {
    for _, logger := range m.loggers {
        logger.Debug(msg, fields)
    }
}

func (m *MultiLoggerAdapter) WithFields(fields port.Fields) port.LoggerPort {
	enrichedLoggers := make([]port.LoggerPort, 0, len(m.loggers))

	for _, logger := range m.loggers {
		enrichedLoggers = append(enrichedLoggers, logger.WithFields(fields))
	}

	return &MultiLoggerAdapter{loggers: enrichedLoggers}
}
