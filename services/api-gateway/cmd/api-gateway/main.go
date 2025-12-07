package main

import (
	"log"
	"api-gateway/internal"
)

func main() {
	application, err := internal.NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("Application run failed: %v", err)
	}
}