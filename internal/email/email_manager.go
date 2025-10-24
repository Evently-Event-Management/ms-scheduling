package email

import (
	"log"

	"ms-scheduling/internal/config"
	"ms-scheduling/internal/models"
)

// EmailSender is an interface for sending emails (to avoid circular dependency)
type EmailSender interface {
	SendEmail(to, subject, body string) error
}

// TemplateGenerator is an interface for generating email templates
type TemplateGenerator interface {
	GenerateSessionCreatedEmail(session *models.EventSession, eventTitle string) EmailTemplate
	GenerateSessionUpdatedEmail(before, after *models.EventSession, eventTitle string) EmailTemplate
	GenerateSessionCancelledEmail(session *models.EventSession, eventTitle string) EmailTemplate
	GenerateSessionReminderEmail(session *models.EventSession, eventTitle string, hoursUntil int) EmailTemplate
	GenerateEventCreatedEmail(event *models.Event, organizationName string) EmailTemplate
	GenerateEventUpdatedEmail(before, after *models.Event, organizationName string) EmailTemplate
	GenerateEventApprovedEmail(event *models.Event, organizationName string) EmailTemplate
	GenerateEventRejectedEmail(event *models.Event, organizationName string) EmailTemplate
	GenerateEventCancelledEmail(event *models.Event, organizationName string) EmailTemplate
	GenerateOrderConfirmedEmail(order interface{}) EmailTemplate
	GenerateOrderPendingEmail(order interface{}) EmailTemplate
	GenerateOrderCancelledEmail(order interface{}) EmailTemplate
	GenerateOrderUpdatedEmail(order interface{}) EmailTemplate
}

// EmailManager centralizes all email sending operations
type EmailManager struct {
	emailSender       EmailSender
	config            config.Config
	templateGenerator TemplateGenerator
}

// NewEmailManager creates a new email manager
func NewEmailManager(emailSender EmailSender, cfg config.Config, templateGen TemplateGenerator) *EmailManager {
	return &EmailManager{
		emailSender:       emailSender,
		config:            cfg,
		templateGenerator: templateGen,
	}
}

// SendEmail sends an email using the provided template
func (m *EmailManager) SendEmail(to string, template EmailTemplate) error {
	log.Printf("[EmailManager] Sending %s email to %s", template.Type.String(), to)

	err := m.emailSender.SendEmail(to, template.Subject, template.HTML)
	if err != nil {
		log.Printf("[EmailManager] Failed to send %s email to %s: %v", template.Type.String(), to, err)
		return err
	}

	log.Printf("[EmailManager] Successfully sent %s email to %s", template.Type.String(), to)
	return nil
}

// Session Email Methods

func (m *EmailManager) SendSessionCreatedEmail(to string, session *models.EventSession, eventTitle string) error {
	template := m.templateGenerator.GenerateSessionCreatedEmail(session, eventTitle)
	return m.SendEmail(to, template)
}

func (m *EmailManager) SendSessionUpdatedEmail(to string, before, after *models.EventSession, eventTitle string) error {
	template := m.templateGenerator.GenerateSessionUpdatedEmail(before, after, eventTitle)
	return m.SendEmail(to, template)
}

func (m *EmailManager) SendSessionCancelledEmail(to string, session *models.EventSession, eventTitle string) error {
	template := m.templateGenerator.GenerateSessionCancelledEmail(session, eventTitle)
	return m.SendEmail(to, template)
}

func (m *EmailManager) SendSessionReminderEmail(to string, session *models.EventSession, eventTitle string, hoursUntil int) error {
	template := m.templateGenerator.GenerateSessionReminderEmail(session, eventTitle, hoursUntil)
	return m.SendEmail(to, template)
}

// Event Email Methods

func (m *EmailManager) SendEventCreatedEmail(to string, event *models.Event, organizationName string) error {
	template := m.templateGenerator.GenerateEventCreatedEmail(event, organizationName)
	return m.SendEmail(to, template)
}

func (m *EmailManager) SendEventUpdatedEmail(to string, before, after *models.Event, organizationName string) error {
	template := m.templateGenerator.GenerateEventUpdatedEmail(before, after, organizationName)
	return m.SendEmail(to, template)
}

func (m *EmailManager) SendEventApprovedEmail(to string, event *models.Event, organizationName string) error {
	template := m.templateGenerator.GenerateEventApprovedEmail(event, organizationName)
	return m.SendEmail(to, template)
}

func (m *EmailManager) SendEventRejectedEmail(to string, event *models.Event, organizationName string) error {
	template := m.templateGenerator.GenerateEventRejectedEmail(event, organizationName)
	return m.SendEmail(to, template)
}

func (m *EmailManager) SendEventCancelledEmail(to string, event *models.Event, organizationName string) error {
	template := m.templateGenerator.GenerateEventCancelledEmail(event, organizationName)
	return m.SendEmail(to, template)
}

// Order Email Methods

func (m *EmailManager) SendOrderConfirmedEmail(to string, order interface{}) error {
	template := m.templateGenerator.GenerateOrderConfirmedEmail(order)
	return m.SendEmail(to, template)
}

func (m *EmailManager) SendOrderPendingEmail(to string, order interface{}) error {
	template := m.templateGenerator.GenerateOrderPendingEmail(order)
	return m.SendEmail(to, template)
}

func (m *EmailManager) SendOrderCancelledEmail(to string, order interface{}) error {
	template := m.templateGenerator.GenerateOrderCancelledEmail(order)
	return m.SendEmail(to, template)
}

func (m *EmailManager) SendOrderUpdatedEmail(to string, order interface{}) error {
	template := m.templateGenerator.GenerateOrderUpdatedEmail(order)
	return m.SendEmail(to, template)
}

// Batch sending methods for multiple recipients

func (m *EmailManager) SendSessionCreatedEmailBatch(subscribers []models.Subscriber, session *models.EventSession, eventTitle string) {
	for _, subscriber := range subscribers {
		if err := m.SendSessionCreatedEmail(subscriber.SubscriberMail, session, eventTitle); err != nil {
			log.Printf("[EmailManager] Failed to send session created email to %s: %v", subscriber.SubscriberMail, err)
		}
	}
}

func (m *EmailManager) SendSessionUpdatedEmailBatch(subscribers []models.Subscriber, before, after *models.EventSession, eventTitle string) {
	for _, subscriber := range subscribers {
		if err := m.SendSessionUpdatedEmail(subscriber.SubscriberMail, before, after, eventTitle); err != nil {
			log.Printf("[EmailManager] Failed to send session updated email to %s: %v", subscriber.SubscriberMail, err)
		}
	}
}

func (m *EmailManager) SendEventCreatedEmailBatch(subscribers []models.Subscriber, event *models.Event, organizationName string) {
	for _, subscriber := range subscribers {
		if err := m.SendEventCreatedEmail(subscriber.SubscriberMail, event, organizationName); err != nil {
			log.Printf("[EmailManager] Failed to send event created email to %s: %v", subscriber.SubscriberMail, err)
		}
	}
}

func (m *EmailManager) SendEventUpdatedEmailBatch(subscribers []models.Subscriber, before, after *models.Event, organizationName string) {
	for _, subscriber := range subscribers {
		if err := m.SendEventUpdatedEmail(subscriber.SubscriberMail, before, after, organizationName); err != nil {
			log.Printf("[EmailManager] Failed to send event updated email to %s: %v", subscriber.SubscriberMail, err)
		}
	}
}
