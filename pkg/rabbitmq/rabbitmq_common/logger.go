package rabbitmq_common

type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(err error, msg string, keysAndValues ...interface{})
}

type noopLogger struct{}

func (l *noopLogger) Debug(msg string, keysAndValues ...interface{})            {}
func (l *noopLogger) Info(msg string, keysAndValues ...interface{})             {}
func (l *noopLogger) Warn(msg string, keysAndValues ...interface{})             {}
func (l *noopLogger) Error(err error, msg string, keysAndValues ...interface{}) {}

func NewNoopLogger() Logger {
	return &noopLogger{}
}
