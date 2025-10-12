package models

import (
	"time"
)

// DebeziumEvent represents the structure of a Debezium change event from Kafka
// for the event_sessions table.
type DebeziumEvent struct {
	Schema  DebeziumSchema  `json:"schema"`
	Payload DebeziumPayload `json:"payload"`
}

// DebeziumSchema represents the schema portion of a Debezium event
type DebeziumSchema struct {
	Type     string                `json:"type"`
	Fields   []DebeziumSchemaField `json:"fields"`
	Optional bool                  `json:"optional"`
	Name     string                `json:"name"`
	Version  int                   `json:"version"`
}

// DebeziumSchemaField represents a field in the Debezium schema
type DebeziumSchemaField struct {
	Type       string                `json:"type"`
	Fields     []DebeziumSchemaField `json:"fields,omitempty"`
	Optional   bool                  `json:"optional"`
	Name       string                `json:"name,omitempty"`
	Version    int                   `json:"version,omitempty"`
	Field      string                `json:"field"`
	Parameters map[string]string     `json:"parameters,omitempty"`
	Default    string                `json:"default,omitempty"`
}

// DebeziumPayload represents the payload portion of a Debezium event
type DebeziumPayload struct {
	Before      *EventSession        `json:"before"`
	After       *EventSession        `json:"after"`
	Source      DebeziumSource       `json:"source"`
	Op          string               `json:"op"`    // c=create, u=update, d=delete, r=read
	TsMs        int64                `json:"ts_ms"` // Unix timestamp in milliseconds
	Transaction *DebeziumTransaction `json:"transaction,omitempty"`
}

// DebeziumSource represents the source metadata in a Debezium event
type DebeziumSource struct {
	Version   string `json:"version"`
	Connector string `json:"connector"`
	Name      string `json:"name"`
	TsMs      int64  `json:"ts_ms"` // Unix timestamp in milliseconds
	Snapshot  string `json:"snapshot,omitempty"`
	DB        string `json:"db"`
	Sequence  string `json:"sequence,omitempty"`
	Schema    string `json:"schema"`
	Table     string `json:"table"`
	TxId      int64  `json:"txId,omitempty"`
	Lsn       int64  `json:"lsn,omitempty"`
	Xmin      *int64 `json:"xmin,omitempty"`
}

// DebeziumTransaction represents transaction information in a Debezium event
type DebeziumTransaction struct {
	ID                  string `json:"id"`
	TotalOrder          int64  `json:"total_order"`
	DataCollectionOrder int64  `json:"data_collection_order"`
}

// EventSession represents the event_sessions table data in a Debezium event
type EventSession struct {
	ID             string `json:"id"`                         // UUID
	EventID        string `json:"event_id"`                   // UUID
	StartTime      int64  `json:"start_time"`                 // Microsecond timestamp
	EndTime        int64  `json:"end_time"`                   // Microsecond timestamp
	Status         string `json:"status"`                     // PENDING, etc.
	VenueDetails   string `json:"venue_details"`              // JSON string
	SessionType    string `json:"session_type"`               // PHYSICAL, ONLINE, etc.
	SalesStartTime int64  `json:"sales_start_time,omitempty"` // Microsecond timestamp
}

// Helper methods to convert Debezium microsecond timestamps to Go time.Time
func MicroTimestampToTime(microTs int64) time.Time {
	return time.Unix(microTs/1000000, (microTs%1000000)*1000)
}

// Helper method to convert Go time.Time to Debezium microsecond timestamp
func TimeToMicroTimestamp(t time.Time) int64 {
	return t.Unix()*1000000 + int64(t.Nanosecond())/1000
}

// SessionUpdate represents a Debezium session update event
type SessionUpdate struct {
	Before    *EventSession  `json:"before"`
	After     *EventSession  `json:"after"`
	Source    DebeziumSource `json:"source"`
	Operation string         `json:"op"` // "c" (create), "u" (update), "d" (delete)
	Timestamp int64          `json:"ts_ms"`
	SessionID string         // Extracted from After.ID or Before.ID
}

// DebeziumSessionEvent represents the full Debezium event structure for sessions
type DebeziumSessionEvent struct {
	Schema  interface{}   `json:"schema"`
	Payload SessionUpdate `json:"payload"`
}

// Event represents the events table data in a Debezium event
type Event struct {
	ID              string `json:"id"`                         // UUID
	OrganizationID  string `json:"organization_id"`            // UUID
	Title           string `json:"title"`                      // Event title
	Description     string `json:"description,omitempty"`      // Event description
	Overview        string `json:"overview,omitempty"`         // Event overview
	Status          string `json:"status"`                     // PENDING, APPROVED, REJECTED, etc.
	RejectionReason string `json:"rejection_reason,omitempty"` // Reason for rejection
	CreatedAt       int64  `json:"created_at"`                 // Microsecond timestamp
	CategoryID      string `json:"category_id,omitempty"`      // UUID
	UpdatedAt       int64  `json:"updated_at,omitempty"`       // Microsecond timestamp
}

// EventUpdate represents a Debezium event update event
type EventUpdate struct {
	Before    *Event         `json:"before"`
	After     *Event         `json:"after"`
	Source    DebeziumSource `json:"source"`
	Operation string         `json:"op"` // "c" (create), "u" (update), "d" (delete)
	Timestamp int64          `json:"ts_ms"`
	EventID   string         // Extracted from After.ID or Before.ID
}

// DebeziumEventEvent represents the full Debezium event structure for events
type DebeziumEventEvent struct {
	Schema  interface{} `json:"schema"`
	Payload EventUpdate `json:"payload"`
}

// Note: Subscription models are defined in subscription.go
