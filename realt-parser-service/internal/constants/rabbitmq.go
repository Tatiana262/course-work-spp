package constants

// Имена очередей
const (
	QueueLinkTasks          = "link_parsing_realt_tasks"
	QueueProcessedProperties = "processed_properties"
)

// Ключи маршрутизации
const (
	RoutingKeyLinkTasks          = "realt.links.tasks"
	RoutingKeyProcessedProperties = "db.properties.save"
)

const (
	FinalDLXExchange   = "link_parsing_tasks_final_dlx"
    FinalDLQ           = "link_parsing_tasks_final_dlq"
    FinalDLQRoutingKey = "links.dlq.key"
)