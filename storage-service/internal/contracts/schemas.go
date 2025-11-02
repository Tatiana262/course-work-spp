package contracts

import (
	// "bytes"
	"encoding/json"
	"fmt"
	"log"

	"github.com/santhosh-tekuri/jsonschema/v5"
)


var compiledSchemas = make(map[string]*jsonschema.Schema)

func init() {
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat = true

	schemaPath := "./schemas/events/processed-real-estate/v1.json"
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		log.Fatalf("failed to compile schema %s: %v", schemaPath, err)
	}

	compiledSchemas["ProcessedRealEstateEvent/1.0.0"] = schema
	log.Println("Successfully loaded schema: ProcessedRealEstateEvent/1.0.0")
}


// ValidateEvent принимает тело сообщения и его метаданные и проверяет по схеме
func ValidateEvent(eventType, eventVersion string, body []byte) error {
	key := fmt.Sprintf("%s/%s", eventType, eventVersion)
	schema, ok := compiledSchemas[key]
	if !ok {
		return fmt.Errorf("schema for event '%s' version '%s' not found", eventType, eventVersion)
	}

	// распарсить JSON в универсальный тип interface{}
	var v interface{}
	if err := json.Unmarshal(body, &v); err != nil {
		// Если это невалидный JSON, валидация по схеме невозможна
		return fmt.Errorf("message body is not a valid JSON: %w", err)
	}

	// валидировать уже распарсенные данные
	if err := schema.Validate(v); err != nil {
		return fmt.Errorf("JSON schema validation failed: %w", err)
	}

	return nil
}