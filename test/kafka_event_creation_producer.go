package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// Define the exact Debezium schema structure from the user's request
type DebeziumEventCreationPayload struct {
	Before      *EventData  `json:"before"`
	After       *EventData  `json:"after"`
	Source      SourceData  `json:"source"`
	Op          string      `json:"op"`
	TsMs        int64       `json:"ts_ms"`
	Transaction interface{} `json:"transaction"`
}

type EventData struct {
	ID              string `json:"id"`
	OrganizationID  string `json:"organization_id"`
	Title           string `json:"title"`
	Description     string `json:"description,omitempty"`
	Overview        string `json:"overview,omitempty"`
	Status          string `json:"status"`
	RejectionReason string `json:"rejection_reason,omitempty"`
	CreatedAt       int64  `json:"created_at"`
	CategoryID      string `json:"category_id,omitempty"`
	UpdatedAt       int64  `json:"updated_at,omitempty"`
}

type SourceData struct {
	Version   string      `json:"version"`
	Connector string      `json:"connector"`
	Name      string      `json:"name"`
	TsMs      int64       `json:"ts_ms"`
	Snapshot  string      `json:"snapshot"`
	DB        string      `json:"db"`
	Sequence  string      `json:"sequence,omitempty"`
	Schema    string      `json:"schema"`
	Table     string      `json:"table"`
	TxId      int64       `json:"txId,omitempty"`
	Lsn       int64       `json:"lsn,omitempty"`
	Xmin      interface{} `json:"xmin,omitempty"`
}

type DebeziumEventCreation struct {
	Schema  interface{}                  `json:"schema"`
	Payload DebeziumEventCreationPayload `json:"payload"`
}

func main() {
	// Kafka configuration
	writer := &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    "dbz.ticketly.public.events",
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	// Create the event creation payload using test parameters: event_id = 456, organization_id = 123
	eventCreation := DebeziumEventCreation{
		Schema: map[string]interface{}{
			"type":     "struct",
			"optional": false,
			"name":     "dbz.ticketly.public.events.Envelope",
			"version":  1,
		},
		Payload: DebeziumEventCreationPayload{
			Before: nil, // No before data for creation
			After: &EventData{
				ID:              "456", // Test event ID
				OrganizationID:  "123", // Test organization ID
				Title:           "An Example Event",
				Description:     "This is a sample event description.",
				Overview:        "An overview of the event goes here.",
				Status:          "PENDING",
				RejectionReason: "",               // null becomes empty string
				CreatedAt:       1759833071227391, // From the example
				CategoryID:      "00363e81-11a7-4daf-8a00-df496d0d2deb",
				UpdatedAt:       1759833071227417, // From the example
			},
			Source: SourceData{
				Version:   "2.5.4.Final",
				Connector: "postgresql",
				Name:      "dbz.ticketly",
				TsMs:      time.Now().UnixMilli(),
				Snapshot:  "false",
				DB:        "event_service",
				Sequence:  "[\"50893648\",\"50893704\"]",
				Schema:    "public",
				Table:     "events",
				TxId:      1219,
				Lsn:       50893704,
				Xmin:      nil,
			},
			Op:          "c",                    // Create operation
			TsMs:        time.Now().UnixMilli(), // Current timestamp in milliseconds
			Transaction: nil,
		},
	}

	// Convert to JSON
	eventJSON, err := json.Marshal(eventCreation)
	if err != nil {
		log.Fatalf("Error marshaling event creation: %v", err)
	}

	log.Printf("Sending event creation for event ID 456 in organization 123 to Kafka...")
	log.Printf("Event JSON: %s", string(eventJSON))

	// Send to Kafka
	msg := kafka.Message{
		Key:   []byte("456"), // Use event ID as key
		Value: eventJSON,
		Time:  time.Now(),
	}

	err = writer.WriteMessages(context.Background(), msg)
	if err != nil {
		log.Fatalf("Error sending event creation to Kafka: %v", err)
	}

	log.Println("✅ Successfully sent event creation notification to Kafka topic dbz.ticketly.public.events")
	log.Printf("Event ID: 456, Organization ID: 123, Status: %s", eventCreation.Payload.After.Status)
}
