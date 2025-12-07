package constants

// Имена очередей
const (
	QueueLinkTasks          = "link_parsing_tasks_kufar"
	QueueSearchTasks			= "tasks_for_search_kufar"
	QueueProcessedProperties = "processed_properties"
)

// Ключи маршрутизации
const (
	RoutingKeyLinkTasks          = "kufar.links.tasks"
	RoutingKeySearchTasks		 = "kufar.search.tasks"
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