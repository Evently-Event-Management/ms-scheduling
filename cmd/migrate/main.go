package main

import (
	"flag"
	"log"
	"ms-scheduling/internal/config"
	"ms-scheduling/internal/services"
	"os"
)

func main() {
	var command = flag.String("command", "up", "Migration command: up, status")
	flag.Parse()

	// Load config
	cfg := config.Load()

	// Initialize database service with DSN directly
	dbService, err := services.NewDatabaseService(cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("Failed to initialize database service: %v", err)
	}
	defer dbService.Close()

	switch *command {
	case "up":
		log.Println("Running migrations...")
		if err := dbService.RunMigrations(); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("✓ Migrations completed successfully")

	case "status":
		log.Println("Checking migration status...")
		if err := dbService.MigrationStatus(); err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}

	default:
		log.Printf("Unknown command: %s", *command)
		log.Println("Available commands: up, status")
		os.Exit(1)
	}
}
