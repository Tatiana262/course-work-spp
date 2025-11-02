package constants

// Имена очередей
const (
	QueueProcessedProperties = "processed_properties"
)

// Ключи маршрутизации
const (
	RoutingKeyProcessedProperties = "db.properties.save"
)


const (
    FinalDLXExchange   = "processed_properties_final_dlx"
    FinalDLQ           = "processed_properties_final_dlq"
    FinalDLQRoutingKey = "properties.dlq.key"
)