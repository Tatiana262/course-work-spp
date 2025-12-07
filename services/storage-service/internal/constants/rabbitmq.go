package constants

// Имена очередей
const (
	QueueProcessedProperties = "processed_properties"
)

// Ключи маршрутизации
const (
	RoutingKeyProcessedProperties = "db.properties.save"
    
    RoutingKeyTaskResults          = "notify.task.result"
)


const (
    FinalDLXExchange   = "processed_properties_final_dlx"
    FinalDLQ           = "processed_properties_final_dlq"
    FinalDLQRoutingKey = "properties.dlq.key"
)