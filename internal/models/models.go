package models

// SQSMessageBody defines the structure of the message we expect.
type SQSMessageBody struct {
	SessionID string `json:"sessionId"`
	Action    string `json:"action"` // e.g., "ON_SALE", "CLOSED"
}
