package configs

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type DBconfig struct {
	URL string
}

type RESTconfig struct {
	PORT string
}

type ApiClientConfig struct {
	STORAGE_PORT string
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
	Rest		RESTconfig
	ApiClient   ApiClientConfig
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
		cfg.AppName = "favorites-service" // Устанавливаем default
	}

	// Читаем DATABASE URL
	cfg.Database.URL = os.Getenv("DATABASE_URL")
	if cfg.Database.URL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	// Читаем конфигурацию для REST
	cfg.Rest.PORT = os.Getenv("PORT")
	if cfg.Rest.PORT == "" {
		cfg.Rest.PORT = "8083"
	}

	cfg.ApiClient.STORAGE_PORT = os.Getenv("STORAGE_SERVICE_URL")
	if cfg.ApiClient.STORAGE_PORT == "" {
		cfg.ApiClient.STORAGE_PORT = "http://localhost:8082"
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