package configs

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"github.com/joho/godotenv"
)

// RabbitMQConfig хранит конфигурацию для RabbitMQ
type RabbitMQConfig struct {
	URL                            string
}

// DBconfig хранит конфигурацию для БД
type DBconfig struct {
	URL string
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

// AppConfig хранит всю конфигурацию приложения
type AppConfig struct {
	AppName   	string 
	Database    DBconfig
	RabbitMQ    RabbitMQConfig 
	FluentBit	FluentBitConfig
	StdoutLogger StdoutLogConfig
}

// LoadConfig загружает конфигурацию из переменных окружения.
func LoadConfig(envPath ...string) (*AppConfig, error) {
	
	var err error
	if len(envPath) > 0 {
		err = godotenv.Load(envPath[0])
	} else {
		err = godotenv.Load()
	}

	if err != nil {
		log.Printf("Info: Could not load .env file (path: %v): %v.\n", envPath, err)
		return nil, fmt.Errorf("сould not load .env file (path: %v): %v", envPath, err)
	}

	cfg := &AppConfig{}

	cfg.AppName = os.Getenv("APP_NAME")
	if cfg.AppName == "" {
		cfg.AppName = "kufar-parser-service" // Устанавливаем default
	}

	// Читаем DATABASE URL
	cfg.Database.URL = os.Getenv("DATABASE_URL")
	if cfg.Database.URL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	// Читаем конфигурацию для RabbitMQ
	cfg.RabbitMQ.URL = os.Getenv("RABBITMQ_URL")
	if cfg.RabbitMQ.URL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL environment variable is required")
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

// getEnvAsString читает переменную окружения как строку или возвращает значение по умолчанию
func getEnvAsString(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt читает переменную окружения как int или возвращает значение по умолчанию
// Логирует ошибку, если переменная есть, но не может быть преобразована в int
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