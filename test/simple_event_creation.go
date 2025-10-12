package main

import (
	"fmt"
	"log"
	"ms-scheduling/internal/config"
	"ms-scheduling/internal/models"
	"ms-scheduling/internal/services"
	"time"
)

func main() {
	log.Println("🚀 Testing event creation email functionality...")

	// Load configuration
	cfg := config.Load()

	// Create email service
	emailService := services.NewEmailService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.FromEmail, cfg.FromName)

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

	// Create a mock subscriber for testing
	userID := "test-user-123"
	testSubscriber := models.Subscriber{
		SubscriberID:   1,
		UserID:         &userID,
		SubscriberMail: "isurumuni.22@cse.mrt.ac.lk", // User's email from conversation
		CreatedAt:      time.Now(),
	}

	// Create the email content
	subject, body := buildTestEventCreationEmail(testSubscriber, eventCreation)

	log.Printf("📧 Sending test email to: %s", testSubscriber.SubscriberMail)
	log.Printf("📧 Subject: %s", subject)
	log.Printf("📧 Body preview: %s", body[:min(200, len(body))]+"...")

	// Send the email
	err := emailService.SendEmail(testSubscriber.SubscriberMail, subject, body)
	if err != nil {
		log.Printf("❌ Error sending email: %v", err)
		return
	}

	log.Println("✅ Event creation notification email sent successfully!")
	log.Println("📧 Check your inbox at isurumuni.22@cse.mrt.ac.lk")
}

// buildTestEventCreationEmail mimics the private method from SubscriberService
func buildTestEventCreationEmail(subscriber models.Subscriber, eventUpdate *models.DebeziumEventEvent) (string, string) {
	after := eventUpdate.Payload.After

	// Convert timestamp to readable format
	timestamp := time.UnixMilli(eventUpdate.Payload.Timestamp)
	createdAt := models.MicroTimestampToTime(after.CreatedAt)

	subject := fmt.Sprintf("🎉 New Event Created: %s", after.Title)

	var body string
	body += fmt.Sprintf("Hello %s,\n\n", subscriber.SubscriberMail)
	body += "🎉 A new event has been created in your subscribed organization!\n\n"

	body += "Event Details:\n"
	body += fmt.Sprintf("• Title: %s\n", after.Title)
	body += fmt.Sprintf("• Status: %s\n", after.Status)

	if after.Description != "" {
		body += fmt.Sprintf("• Description: %s\n", after.Description)
	}

	if after.Overview != "" {
		body += fmt.Sprintf("• Overview: %s\n", after.Overview)
	}

	body += fmt.Sprintf("• Created: %s\n", createdAt.Format("2006-01-02 15:04:05"))
	body += fmt.Sprintf("• Event ID: %s\n", after.ID)
	body += fmt.Sprintf("• Organization ID: %s\n", after.OrganizationID)

	if after.CategoryID != "" {
		body += fmt.Sprintf("• Category ID: %s\n", after.CategoryID)
	}

	body += fmt.Sprintf("\n📅 Notification sent at: %s\n", timestamp.Format("2006-01-02 15:04:05"))

	if after.Status == "PENDING" {
		body += "\n⏳ This event is currently pending approval. You'll be notified when it's approved and ready for booking.\n"
	} else if after.Status == "APPROVED" {
		body += "\n✅ This event is approved and ready for booking!\n"
	}

	body += "\nStay tuned for more updates about this event!"
	body += "\n\nBest regards,\nTicketly Team"

	return subject, body
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
