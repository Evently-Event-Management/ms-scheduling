package main

import (
	"database/sql"
	"fmt"
	"log"
	"ms-scheduling/internal/config"
	"ms-scheduling/internal/models"
	"ms-scheduling/internal/services"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("🚀 Starting comprehensive event creation notification test...")

	// Load configuration
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

	// Get database connection
	db := dbService.DB
	log.Println("✅ Database connection successful")

	// Setup test data - organization subscriptions
	log.Println("📋 Setting up organization subscriptions for testing...")
	if err := setupOrganizationSubscriptions(db); err != nil {
		log.Fatalf("Error setting up test data: %v", err)
	}
	log.Println("✅ Test organization subscriptions created")

	// Create services
	keycloakClient := services.NewKeycloakClient(cfg.KeycloakURL, cfg.KeycloakRealm, cfg.ClientID, cfg.ClientSecret)
	emailService := services.NewEmailService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.FromEmail, cfg.FromName)
	subscriberService := services.NewSubscriberService(db, keycloakClient, emailService)

	// Test 1: Direct event creation notification
	log.Println("\n🧪 Test 1: Direct event creation notification processing...")
	if err := testDirectEventCreation(subscriberService); err != nil {
		log.Printf("❌ Direct test failed: %v", err)
	} else {
		log.Println("✅ Direct event creation test passed!")
	}

	// Verify subscribers count
	log.Println("\n📊 Verifying organization subscribers...")
	subscribers, err := subscriberService.GetOrganizationSubscribers("123")
	if err != nil {
		log.Printf("❌ Error getting organization subscribers: %v", err)
	} else {
		log.Printf("✅ Found %d subscribers for organization 123:", len(subscribers))
		for _, sub := range subscribers {
			log.Printf("   - %s (ID: %d)", sub.SubscriberMail, sub.SubscriberID)
		}
	}

	log.Println("\n🎉 Comprehensive event creation notification test completed!")
	log.Println("📧 Check your email inbox at isurumuni.22@cse.mrt.ac.lk for the event creation notification!")
}

func setupOrganizationSubscriptions(db *sql.DB) error {
	// Insert organization subscriptions for test organization ID 123
	subscriptions := []struct {
		subscriberID int
		targetID     int
	}{
		{1, 123}, // isurumuni.22@cse.mrt.ac.lk
		{2, 123}, // user2@example.com
		{3, 123}, // user3@example.com
		{4, 123}, // customer@example.com
	}

	for _, sub := range subscriptions {
		query := `
			INSERT INTO subscriptions (subscriber_id, category, target_id, created_at) 
			SELECT $1, 'organization', $2, NOW()
			WHERE NOT EXISTS (
				SELECT 1 FROM subscriptions 
				WHERE subscriber_id = $1 AND category = 'organization' AND target_id = $2
			)
		`
		_, err := db.Exec(query, sub.subscriberID, sub.targetID)
		if err != nil {
			return fmt.Errorf("error inserting organization subscription for subscriber %d: %w", sub.subscriberID, err)
		}
	}

	return nil
}

func testDirectEventCreation(subscriberService *services.SubscriberService) error {
	// Create test event creation notification matching the Debezium schema provided
	eventCreation := &models.DebeziumEventEvent{
		Payload: models.EventUpdate{
			Before: nil, // No before data for creation
			After: &models.Event{
				ID:              "456",                                   // Test event ID from user request
				OrganizationID:  "123",                                   // Test organization ID from user request
				Title:           "An Example Event",                      // From the provided schema
				Description:     "This is a sample event description.",   // From the provided schema
				Overview:        "An overview of the event goes here.",   // From the provided schema
				Status:          "PENDING",                               // From the provided schema
				RejectionReason: "",                                      // null becomes empty string
				CreatedAt:       models.TimeToMicroTimestamp(time.Now()), // Current time as microseconds
				CategoryID:      "00363e81-11a7-4daf-8a00-df496d0d2deb",  // From the provided schema
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
			EventID:   "456",                  // Test event ID
		},
	}

	log.Printf("Processing event creation for event %s in organization %s",
		eventCreation.Payload.After.ID, eventCreation.Payload.After.OrganizationID)

	// Process the event creation notification
	return subscriberService.ProcessEventCreation(eventCreation)
}
