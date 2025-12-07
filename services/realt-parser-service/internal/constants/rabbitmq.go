package constants

// Имена очередей
const (
	QueueLinkTasks          = "link_parsing_tasks_realt"
	QueueSearchTasks			= "tasks_for_search_realt"
	QueueProcessedProperties = "processed_properties"
)

// Ключи маршрутизации
const (
	RoutingKeyLinkTasks          = "realt.links.tasks"
	RoutingKeySearchTasks		 = "realt.search.tasks"
	RoutingKeyProcessedProperties = "db.properties.save"
	RoutingKeyTaskResults          = "notify.task.result"
)

const (
	FinalDLXExchange   = "link_parsing_tasks_final_dlx"
    FinalDLQ           = "link_parsing_tasks_final_dlq"
    FinalDLQRoutingKey = "links.dlq.key"

	FinalDLXExchangeForSearchTasks = "search_tasks_final_dlx"
	FinalDLQForSearchTasks = "search_tasks_final_dlq"
	FinalDLQRoutingKeyForSearchTasks = "search_tasks.dlq.key"
)