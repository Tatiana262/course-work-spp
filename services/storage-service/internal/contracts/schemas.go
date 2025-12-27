package contracts

import (
	// "bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"strings"

	"real-estate-system/schemas"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)



var compiledSchemas = make(map[string]*jsonschema.Schema)


func init() {
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat = true

	// Добавляем все схемы как ресурсы
	// Это нужно, чтобы схемы могли ссылаться друг на друга через `$ref`
	err := fs.WalkDir(schemas.SchemasFS, "events", func(path string, d fs.DirEntry, err error) error {
		if err != nil { return err }
		if !d.IsDir() && strings.HasSuffix(path, ".json") {
			file, _ := schemas.SchemasFS.Open(path)
			defer file.Close()
			if err := compiler.AddResource(path, file); err != nil { // Добавляем префикс, чтобы URL был валидным
				log.Fatalf("failed to add schema resource %s: %v", path, err)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatalf("error walking and adding schema resources: %v", err)
	}

	// Снова обходим для компиляции и регистрации
	err = fs.WalkDir(schemas.SchemasFS, "events", func(path string, d fs.DirEntry, err error) error {
		if err != nil { return err }
		if !d.IsDir() && strings.HasSuffix(path, ".json") {
			
			schema, err := compiler.Compile(path)
			if err != nil {
				log.Printf("WARNING: could not compile schema %s: %v. Skipping.", path, err)
				return nil 
			}
			
			key := generateKeyFromPath(path)
			compiledSchemas[key] = schema
			log.Printf("Successfully compiled and registered schema: %s -> %s", path, key)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("error walking and compiling schemas: %v", err)
	}
}

// generateKeyFromPath преобразует путь вида "schemas/events/processed-real-estate/v1.json"
// в ключ вида "ProcessedRealEstateEvent/1.0.0".
func generateKeyFromPath(path string) string {
	
	trimmedPath := strings.TrimPrefix(path, "events/")
	trimmedPath = strings.TrimSuffix(trimmedPath, ".json")
	
	parts := strings.Split(trimmedPath, "/")
	if len(parts) != 2 {
		return "" // Некорректный путь, возвращаем пустой ключ
	}
	
	caser := cases.Title(language.English)

	eventNameParts := strings.Split(parts[0], "-")
	var eventNameBuilder strings.Builder
	for _, p := range eventNameParts {
		eventNameBuilder.WriteString(caser.String(p))
	}
	eventNameBuilder.WriteString("Event")
	eventName := eventNameBuilder.String()
	
	version := strings.Replace(parts[1], "v", "", 1) + ".0.0"
	
	return fmt.Sprintf("%s/%s", eventName, version)
}


// ValidateEvent принимает тело сообщения и его метаданные и проверяет по схеме
func ValidateEvent(eventType, eventVersion string, body []byte) error {
	key := fmt.Sprintf("%s/%s", eventType, eventVersion)
	schema, ok := compiledSchemas[key]
	if !ok {
		return fmt.Errorf("schema for event '%s' version '%s' not found", eventType, eventVersion)
	}

	// Распарсить JSON в универсальный тип interface{}
	var v interface{}
	if err := json.Unmarshal(body, &v); err != nil {
		// Если это невалидный JSON, валидация по схеме невозможна
		return fmt.Errorf("message body is not a valid JSON: %w", err)
	}

	// Валидировать уже распарсенные данные
	if err := schema.Validate(v); err != nil {
		// Возвращаем подробную ошибку валидации
		return fmt.Errorf("JSON schema validation failed: %w", err)
	}

	return nil
}