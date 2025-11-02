package configs

import (
	"fmt"
	"log"
	"os"

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

type RESTconfig struct {
	PORT string
}

// AppConfig хранит всю конфигурацию приложения
type AppConfig struct {
	Database    DBconfig
	RabbitMQ    RabbitMQConfig 
	Rest		RESTconfig
}

// LoadConfig загружает конфигурацию из переменных окружения
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

	// Читаем конфигурацию для REST
	cfg.Rest.PORT = os.Getenv("PORT")
	if cfg.Rest.PORT == "" {
		cfg.Rest.PORT = "8080"
	}

	return cfg, nil
}