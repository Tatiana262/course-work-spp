package constants

// Имена очередей
const (
	QueueTaskResults          = "task_results"
	// QueueTaskCompletionResults = "task_completion_results"
)

// Ключи маршрутизации
const (
	RoutingKeyTaskResults          = "notify.task.result"
	// RoutingKeyTaskCompletionResults = "task.completion.results"
)

const (
	FinalDLXExchange   = "task_results_final_dlx"
    FinalDLQ           = "task_results_final_dlq"
    FinalDLQRoutingKey = "task_results.dlq.key"
)

const MainExchange = "main_exchange"

const (
	RetryExchange = "shared_retry_exchange"
	WaitQueue     = "shared_wait_10s"
	RetryTTL      = 10000 // 10 секунд
)

const (
	LinkParsingTasksDlq = "link_parsing_tasks_final_dlq"
	TasksForSearchDlq = "tasks_for_search_final_dlq"
	ProcessedPropertiesDlq = "processed_properties_final_dlq"
)