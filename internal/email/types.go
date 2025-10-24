package email

// EmailCategory represents the main category of the email
type EmailCategory string

const (
	CategorySession      EmailCategory = "SESSION"
	CategoryEvent        EmailCategory = "EVENT"
	CategoryOrganization EmailCategory = "ORGANIZATION"
	CategoryPayment      EmailCategory = "PAYMENT"
	CategoryOrder        EmailCategory = "ORDER"
)

// EmailAction represents the action that triggered the email
type EmailAction string

const (
	ActionCreated   EmailAction = "CREATED"
	ActionUpdated   EmailAction = "UPDATED"
	ActionCancelled EmailAction = "CANCELLED"
	ActionDeleted   EmailAction = "DELETED"
	ActionApproved  EmailAction = "APPROVED"
	ActionRejected  EmailAction = "REJECTED"
	ActionConfirmed EmailAction = "CONFIRMED"
	ActionPending   EmailAction = "PENDING"
	ActionSuccess   EmailAction = "SUCCESS"
	ActionFailed    EmailAction = "FAILED"
	ActionRefunded  EmailAction = "REFUNDED"
	ActionReminder  EmailAction = "REMINDER"
)

// EmailType represents a specific type of email combining category and action
type EmailType struct {
	Category EmailCategory
	Action   EmailAction
}

// Common email types
var (
	// Session emails
	EmailSessionCreated   = EmailType{CategorySession, ActionCreated}
	EmailSessionUpdated   = EmailType{CategorySession, ActionUpdated}
	EmailSessionCancelled = EmailType{CategorySession, ActionCancelled}
	EmailSessionDeleted   = EmailType{CategorySession, ActionDeleted}
	EmailSessionReminder  = EmailType{CategorySession, ActionReminder}

	// Event emails
	EmailEventCreated   = EmailType{CategoryEvent, ActionCreated}
	EmailEventUpdated   = EmailType{CategoryEvent, ActionUpdated}
	EmailEventCancelled = EmailType{CategoryEvent, ActionCancelled}
	EmailEventApproved  = EmailType{CategoryEvent, ActionApproved}
	EmailEventRejected  = EmailType{CategoryEvent, ActionRejected}

	// Organization emails
	EmailOrganizationCreated   = EmailType{CategoryOrganization, ActionCreated}
	EmailOrganizationUpdated   = EmailType{CategoryOrganization, ActionUpdated}
	EmailOrganizationApproved  = EmailType{CategoryOrganization, ActionApproved}
	EmailOrganizationRejected  = EmailType{CategoryOrganization, ActionRejected}
	EmailOrganizationCancelled = EmailType{CategoryOrganization, ActionCancelled}

	// Order emails
	EmailOrderConfirmed = EmailType{CategoryOrder, ActionConfirmed}
	EmailOrderPending   = EmailType{CategoryOrder, ActionPending}
	EmailOrderCancelled = EmailType{CategoryOrder, ActionCancelled}
	EmailOrderUpdated   = EmailType{CategoryOrder, ActionUpdated}

	// Payment emails
	EmailPaymentSuccess  = EmailType{CategoryPayment, ActionSuccess}
	EmailPaymentFailed   = EmailType{CategoryPayment, ActionFailed}
	EmailPaymentPending  = EmailType{CategoryPayment, ActionPending}
	EmailPaymentRefunded = EmailType{CategoryPayment, ActionRefunded}
)

// EmailTemplate represents a complete email template with subject and body
type EmailTemplate struct {
	Type    EmailType
	Subject string
	HTML    string
	Text    string // Plain text version (optional)
}

// String returns a string representation of the email type
func (et EmailType) String() string {
	return string(et.Category) + "_" + string(et.Action)
}
