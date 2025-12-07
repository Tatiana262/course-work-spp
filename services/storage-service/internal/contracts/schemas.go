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

// init теперь полностью автоматический.
func init() {
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat = true

	// --- ШАГ 1: Добавляем все схемы как ресурсы (как и раньше) ---
	// Это нужно, чтобы схемы могли ссылаться друг на друга через `$ref`.
	// Путь внутри embed FS начинается с 'schemas'.
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

	// --- ШАГ 2: Снова обходим, но теперь для КОМПИЛЯЦИИ и РЕГИСТРАЦИИ ---
	err = fs.WalkDir(schemas.SchemasFS, "events", func(path string, d fs.DirEntry, err error) error {
		if err != nil { return err }
		if !d.IsDir() && strings.HasSuffix(path, ".json") {
			
			// schema, err := compiler.Compile("file:///" + path)
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
	// 1. Убираем "schemas/events/" и ".json"
	// "schemas/events/processed-real-estate/v1.json" -> "processed-real-estate/v1"
	trimmedPath := strings.TrimPrefix(path, "events/")
	trimmedPath = strings.TrimSuffix(trimmedPath, ".json")
	
	parts := strings.Split(trimmedPath, "/")
	if len(parts) != 2 {
		return "" // Некорректный путь, возвращаем пустой ключ
	}
	
	caser := cases.Title(language.English)

	// 2. Преобразуем "processed-real-estate" в "ProcessedRealEstateEvent"
	eventNameParts := strings.Split(parts[0], "-")
	var eventNameBuilder strings.Builder
	for _, p := range eventNameParts {
		eventNameBuilder.WriteString(caser.String(p))
	}
	eventNameBuilder.WriteString("Event")
	eventName := eventNameBuilder.String()
	
	// 3. Преобразуем "v1" в "1.0.0"
	version := strings.Replace(parts[1], "v", "", 1) + ".0.0"
	
	return fmt.Sprintf("%s/%s", eventName, version)
}


// ValidateEvent принимает тело сообщения и его метаданные и проверяет по схеме.
func ValidateEvent(eventType, eventVersion string, body []byte) error {
	key := fmt.Sprintf("%s/%s", eventType, eventVersion)
	schema, ok := compiledSchemas[key]
	if !ok {
		return fmt.Errorf("schema for event '%s' version '%s' not found", eventType, eventVersion)
	}

	// ШАГ 1: Распарсить JSON в универсальный тип interface{}
	var v interface{}
	if err := json.Unmarshal(body, &v); err != nil {
		// Если это невалидный JSON, валидация по схеме невозможна
		return fmt.Errorf("message body is not a valid JSON: %w", err)
	}

	// ШАГ 2: Валидировать уже распарсенные данные
	if err := schema.Validate(v); err != nil {
		// Возвращаем подробную ошибку валидации
		return fmt.Errorf("JSON schema validation failed: %w", err)
	}

	return nil
}