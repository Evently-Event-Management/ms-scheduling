package main

import (
	"database/sql"
	"log"
	"ms-scheduling/internal/config"
	"ms-scheduling/internal/models"
	"ms-scheduling/internal/services"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}

	// Create services
	keycloakClient := services.NewKeycloakClient(cfg.KeycloakURL)
	emailService := services.NewEmailService(cfg.SMTPConfig)
	subscriberService := services.NewSubscriberService(db, keycloakClient, emailService)

	// Create test event creation notification
	// Using the provided test parameters: event_id = 456, organization_id = 123
	eventCreation := &models.DebeziumEventEvent{
		Payload: models.EventUpdate{
			Before: nil, // No before data for creation
			After: &models.Event{
				ID:              "456",                                   // Event ID from request
				OrganizationID:  "123",                                   // Organization ID from request
				Title:           "An Example Event",                      // From the schema example
				Description:     "This is a sample event description.",   // From the schema example
				Overview:        "An overview of the event goes here.",   // From the schema example
				Status:          "PENDING",                               // From the schema example
				RejectionReason: "",                                      // null in schema
				CreatedAt:       models.TimeToMicroTimestamp(time.Now()), // Current time as microseconds
				CategoryID:      "00363e81-11a7-4daf-8a00-df496d0d2deb",  // From the schema example
				UpdatedAt:       models.TimeToMicroTimestamp(time.Now()), // Current time as microseconds
			},
			Source: models.DebeziumSource{
				Version:   "2.5.4.Final",
				Connector: "postgresql",
				Name:      "dbz.ticketly",
				TsMs:      time.Now().UnixMilli(),
				Snapshot:  "false",
				DB:        "event_service",
				Schema:    "public",
				Table:     "events",
			},
			Operation: "c",                    // Create operation
			Timestamp: time.Now().UnixMilli(), // Current timestamp
			EventID:   "456",                  // Event ID from request
		},
	}

	log.Printf("Testing event creation notification for event %s in organization %s",
		eventCreation.Payload.After.ID, eventCreation.Payload.After.OrganizationID)

	// Process the event creation notification
	err = subscriberService.ProcessEventCreation(eventCreation)
	if err != nil {
		log.Printf("Error processing event creation notification: %v", err)
		return
	}

	log.Println("✅ Event creation notification test completed successfully!")
}
