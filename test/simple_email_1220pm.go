package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// SimpleEmailMessage represents a simplified message for email notifications only
type SimpleEmailMessage struct {
	Schema  SimpleSchema  `json:"schema"`
	Payload SimplePayload `json:"payload"`
}

type SimpleSchema struct {
	Type     string `json:"type"`
	Optional bool   `json:"optional"`
	Name     string `json:"name"`
	Version  int    `json:"version"`
}

type SimplePayload struct {
	Before      interface{}  `json:"before"`
	After       SimpleAfter  `json:"after"`
	Source      SimpleSource `json:"source"`
	Op          string       `json:"op"`
	TsMs        int64        `json:"ts_ms"`
	Transaction interface{}  `json:"transaction"`
}

type SimpleAfter struct {
	ID             string `json:"id"`
	EventID        string `json:"event_id"`
	StartTime      int64  `json:"start_time"`
	EndTime        int64  `json:"end_time"`
	Status         string `json:"status"`
	VenueDetails   string `json:"venue_details"`
	SessionType    string `json:"session_type"`
	SalesStartTime int64  `json:"sales_start_time"`
}

type SimpleSource struct {
	Version   string `json:"version"`
	Connector string `json:"connector"`
	Name      string `json:"name"`
	TsMs      int64  `json:"ts_ms"`
	Snapshot  string `json:"snapshot"`
	Db        string `json:"db"`
	Schema    string `json:"schema"`
	Table     string `json:"table"`
}

// generateUUID generates a simple UUID v4
func generateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp-based ID if random fails
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}

	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant bits

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func main() {
	fmt.Println("📧 Simple Email Producer - Tomorrow 2:15 PM")
	fmt.Println("============================================")

	// Calculate tomorrow 2:15 PM
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	emailTime := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 14, 15, 0, 0, tomorrow.Location())

	fmt.Printf("Current time: %s\n", now.Format("2006-01-02 15:04:05"))
	fmt.Printf("Email scheduled for: %s\n", emailTime.Format("2006-01-02 15:04:05"))

	// Generate UUIDs
	sessionID := generateUUID()
	eventID := generateUUID()
	fmt.Printf(sessionID)
	fmt.Printf(eventID)

	// Create simplified message
	message := SimpleEmailMessage{
		Schema: SimpleSchema{
			Type:     "struct",
			Optional: false,
			Name:     "dbz.ticketly.public.event_sessions.Envelope",
			Version:  1,
		},
		Payload: SimplePayload{
			Before: nil,
			After: SimpleAfter{
				ID:             sessionID,
				EventID:        eventID,
				StartTime:      emailTime.UnixMicro(),
				EndTime:        emailTime.Add(2 * time.Hour).UnixMicro(),
				Status:         "PENDING",
				VenueDetails:   `{"name": "Email Test Event", "address": "Test Location"}`,
				SessionType:    "PHYSICAL",
				SalesStartTime: emailTime.Add(-1 * time.Hour).UnixMicro(), // 1 hour before start
			},
			Source: SimpleSource{
				Version:   "2.5.4.Final",
				Connector: "postgresql",
				Name:      "dbz.ticketly",
				TsMs:      time.Now().UnixMilli(),
				Snapshot:  "false",
				Db:        "event_service",
				Schema:    "public",
				Table:     "event_sessions",
			},
			Op:          "c",
			TsMs:        time.Now().UnixMilli(),
			Transaction: nil,
		},
	}

	// Convert to JSON
	messageBytes, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal message: %v", err)
	}

	fmt.Println("\n📄 Email Notification Message:")
	fmt.Println("===============================")
	fmt.Printf("Session ID: %s\n", sessionID)
	fmt.Printf("Event ID: %s\n", eventID)
	fmt.Printf("Start Time: %s\n", emailTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Microseconds: %d\n", emailTime.UnixMicro())

	// Print compact JSON for logs
	compactBytes, _ := json.Marshal(message)
	fmt.Println("\n📋 Compact JSON:")
	fmt.Println(string(compactBytes))

	// Send to Kafka
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{"localhost:9092"},
		Topic:    "dbz.ticketly.public.event_sessions",
		Balancer: &kafka.LeastBytes{},
	})
	defer writer.Close()

	kafkaMsg := kafka.Message{
		Key:   []byte(sessionID),
		Value: messageBytes,
		Headers: []kafka.Header{
			{Key: "source", Value: []byte("email-test")},
			{Key: "operation", Value: []byte("create")},
			{Key: "session_type", Value: []byte("email_notification")},
		},
	}

	fmt.Println("\n🚀 Sending to Kafka...")
	err = writer.WriteMessages(context.Background(), kafkaMsg)
	if err != nil {
		log.Printf("❌ Kafka error: %v", err)
		log.Println("💡 Ensure Kafka is running on localhost:9092")
		return
	}

	fmt.Println("✅ Message sent successfully!")
	fmt.Printf("🎯 Target: %s at %s\n", sessionID, emailTime.Format("2006-01-02 15:04:05"))
	fmt.Println("📧 Email notification should be scheduled!")
	fmt.Println("\n🔍 Check your scheduling service logs to see the processing...")
}
