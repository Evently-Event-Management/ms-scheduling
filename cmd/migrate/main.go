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

	// Initialize database service
	dbConfig := services.DatabaseConfig{
		Host:     cfg.DatabaseHost,
		Port:     cfg.DatabasePort,
		User:     cfg.DatabaseUser,
		Password: cfg.DatabasePassword,
		DBName:   cfg.DatabaseName,
		SSLMode:  cfg.DatabaseSSLMode,
	}

	dbService, err := services.NewDatabaseService(dbConfig)
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
