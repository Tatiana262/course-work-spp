package constants

// Имена очередей
const (
	QueueTaskResults          = "task_results"
	QueueTaskCompletionResults = "task_completion_results"
)

// Ключи маршрутизации
const (
	RoutingKeyTaskResults          = "notify.task.result"
	RoutingKeyTaskCompletionResults = "task.completion.results"
)

const (
	FinalDLXExchange   = "task_results_final_dlx"
    FinalDLQ           = "task_results_final_dlq"
    FinalDLQRoutingKey = "task_results.dlq.key"
)