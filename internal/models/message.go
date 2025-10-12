package models

// SQSMessageBody represents the standard scheduling message body format
type SQSMessageBody struct {
	SessionID string `json:"session_id"`
	Action    string `json:"action"`
}

// SQSReminderMessageBody represents the reminder-specific message body format
type SQSReminderMessageBody struct {
	SessionID      string `json:"session_id"`
	Action         string `json:"action"`
	ReminderType   string `json:"reminder_type"`
	TemplateID     string `json:"template_id,omitempty"`
	NotificationID string `json:"notification_id,omitempty"`
}
