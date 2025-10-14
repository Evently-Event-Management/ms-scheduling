package models

// SQSMessageBody represents the standard scheduling message body format
type SQSMessageBody struct {
	SessionID string `json:"session_id"`
	Action    string `json:"action"`
}

// SQSReminderMessageBody represents the reminder-specific message body format
// The ReminderType field replaces the need for Action - use only ReminderType for logic
type SQSReminderMessageBody struct {
	SessionID      string `json:"session_id"`
	ReminderType   string `json:"reminder_type"`
	TemplateID     string `json:"template_id,omitempty"`
	NotificationID string `json:"notification_id,omitempty"`
}
