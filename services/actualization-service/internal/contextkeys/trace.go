package contextkeys

import (
	"context"
)

// Тип для ключа контекста
type traceIDKeyType struct{}

var traceIDKey = traceIDKeyType{}

// ContextWithTraceID помещает trace_id в контекст
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceIDFromContext извлекает trace_id из контекста
// Возвращает пустую строку, если trace_id не найден
func TraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}