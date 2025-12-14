package configs

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config хранит всю конфигурацию приложения.
type Config struct {
	Port string // Порт, на котором будет работать сам Gateway

	// URL-адреса внутренних сервисов
	AuthServiceURL          string
	StorageServiceURL       string
	FavoritesServiceURL     string
	ActualizationServiceURL string
	TasksServiceURL  		string

	FluentBit	FluentBitConfig
	StdoutLogger StdoutLogConfig
	AppName   	string 
}

type StdoutLogConfig struct {
    Level string `mapstructure:"STDOUT_LOG_LEVEL" default:"debug"` // По умолчанию DEBUG
}

type FluentBitConfig struct {
	Host string
	Port int
	Enabled bool
	Level   string `mapstructure:"FLUENTBIT_LOG_LEVEL" default:"info"` // По умолчанию INFO
}

// LoadConfig загружает конфигурацию из переменных окружения.
// Рекомендуется использовать .env файл для локальной разработки.
func LoadConfig(envPath ...string) (*Config, error) {
	var err error
	if len(envPath) > 0 {
		err = godotenv.Load(envPath[0])
	} else {
		err = godotenv.Load()
	}

	if err != nil {
		log.Println("No .env file found, using environment variables")
		return nil, fmt.Errorf("сould not load .env file (path: %v): %v", envPath, err)
	}

	cfg := &Config{
		Port: getEnv("GATEWAY_PORT", "8080"),

		AuthServiceURL:          getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		StorageServiceURL:       getEnv("STORAGE_SERVICE_URL", "http://localhost:8082"),
		FavoritesServiceURL:     getEnv("FAVORITES_SERVICE_URL", "http://localhost:8083"),
		ActualizationServiceURL: getEnv("ACTUALIZATION_SERVICE_URL", "http://localhost:8084"),
		TasksServiceURL:         getEnv("TASKS_SERVICE_URL", "http://localhost:8084"),
		AppName: 				 getEnv("APP_NAME", "api-gateway"),
	}

	cfg.FluentBit.Host = os.Getenv("FLUENTBIT_HOST")
	if cfg.FluentBit.Host == "" {
		return nil, fmt.Errorf("FLUENTBIT_HOST environment variable is required")
	}

	cfg.FluentBit.Enabled = getEnvAsBool("FLUENTBIT_ENABLED", false)
	if cfg.FluentBit.Enabled {
		cfg.FluentBit.Host = os.Getenv("FLUENTBIT_HOST")
		if cfg.FluentBit.Host == "" {
			log.Println("WARNING: FLUENTBIT_ENABLED is true, but FLUENTBIT_HOST is not set. Disabling Fluent Bit.")
			cfg.FluentBit.Enabled = false
		}

		cfg.FluentBit.Port = getEnvAsInt("FLUENTBIT_PORT", 24224)
		cfg.FluentBit.Level = getEnvAsString("FLUENTBIT_LOG_LEVEL", "info")
	}

	cfg.StdoutLogger.Level = getEnvAsString("STDOUT_LOG_LEVEL", "debug")

	return cfg, nil
}

func getEnvAsString(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnv - вспомогательная функция для чтения переменных окружения с значением по умолчанию.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}


func getEnvAsInt(key string, defaultValue int) int {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	valueInt, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Warning: Environment variable %s (value: %s) could not be parsed as int: %v. Using default value: %d\n", key, valueStr, err, defaultValue)
		return defaultValue
	}
	return valueInt
}

// getEnvAsBool читает переменную окружения как bool или возвращает значение по умолчанию
func getEnvAsBool(key string, defaultValue bool) bool {
	valStr, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	valBool, err := strconv.ParseBool(valStr)
	if err != nil {
		log.Printf("Warning: Environment variable %s (value: %s) could not be parsed as bool: %v. Using default value: %t\n", key, valStr, err, defaultValue)
		return defaultValue
	}
	return valBool
}