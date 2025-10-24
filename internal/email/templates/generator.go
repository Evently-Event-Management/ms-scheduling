package templates

import (
	"ms-scheduling/internal/email"
	"ms-scheduling/internal/models"
)

// StandardTemplateGenerator implements email.TemplateGenerator
type StandardTemplateGenerator struct{}

// NewStandardTemplateGenerator creates a new standard template generator
func NewStandardTemplateGenerator() *StandardTemplateGenerator {
	return &StandardTemplateGenerator{}
}

// Session templates
func (g *StandardTemplateGenerator) GenerateSessionCreatedEmail(session *models.EventSession, eventTitle string) email.EmailTemplate {
	return GenerateSessionCreatedEmail(session, eventTitle)
}

func (g *StandardTemplateGenerator) GenerateSessionUpdatedEmail(before, after *models.EventSession, eventTitle string) email.EmailTemplate {
	return GenerateSessionUpdatedEmail(before, after, eventTitle)
}

func (g *StandardTemplateGenerator) GenerateSessionCancelledEmail(session *models.EventSession, eventTitle string) email.EmailTemplate {
	return GenerateSessionCancelledEmail(session, eventTitle)
}

func (g *StandardTemplateGenerator) GenerateSessionReminderEmail(session *models.EventSession, eventTitle string, hoursUntil int) email.EmailTemplate {
	return GenerateSessionReminderEmail(session, eventTitle, hoursUntil)
}

// Event templates
func (g *StandardTemplateGenerator) GenerateEventCreatedEmail(event *models.Event, organizationName string) email.EmailTemplate {
	return GenerateEventCreatedEmail(event, organizationName)
}

func (g *StandardTemplateGenerator) GenerateEventUpdatedEmail(before, after *models.Event, organizationName string) email.EmailTemplate {
	return GenerateEventUpdatedEmail(before, after, organizationName)
}

func (g *StandardTemplateGenerator) GenerateEventApprovedEmail(event *models.Event, organizationName string) email.EmailTemplate {
	return GenerateEventApprovedEmail(event, organizationName)
}

func (g *StandardTemplateGenerator) GenerateEventRejectedEmail(event *models.Event, organizationName string) email.EmailTemplate {
	return GenerateEventRejectedEmail(event, organizationName)
}

func (g *StandardTemplateGenerator) GenerateEventCancelledEmail(event *models.Event, organizationName string) email.EmailTemplate {
	return GenerateEventCancelledEmail(event, organizationName)
}

// Order templates
func (g *StandardTemplateGenerator) GenerateOrderConfirmedEmail(order interface{}) email.EmailTemplate {
	orderData, ok := order.(*OrderData)
	if !ok {
		return email.EmailTemplate{}
	}
	return GenerateOrderConfirmedEmail(orderData)
}

func (g *StandardTemplateGenerator) GenerateOrderPendingEmail(order interface{}) email.EmailTemplate {
	orderData, ok := order.(*OrderData)
	if !ok {
		return email.EmailTemplate{}
	}
	return GenerateOrderPendingEmail(orderData)
}

func (g *StandardTemplateGenerator) GenerateOrderCancelledEmail(order interface{}) email.EmailTemplate {
	orderData, ok := order.(*OrderData)
	if !ok {
		return email.EmailTemplate{}
	}
	return GenerateOrderCancelledEmail(orderData)
}

func (g *StandardTemplateGenerator) GenerateOrderUpdatedEmail(order interface{}) email.EmailTemplate {
	orderData, ok := order.(*OrderData)
	if !ok {
		return email.EmailTemplate{}
	}
	return GenerateOrderUpdatedEmail(orderData)
}

// Payment templates
func (g *StandardTemplateGenerator) GeneratePaymentSuccessEmail(payment interface{}) email.EmailTemplate {
	paymentData, ok := payment.(*PaymentData)
	if !ok {
		return email.EmailTemplate{}
	}
	return GeneratePaymentSuccessEmail(paymentData)
}

func (g *StandardTemplateGenerator) GeneratePaymentFailedEmail(payment interface{}, reason string) email.EmailTemplate {
	paymentData, ok := payment.(*PaymentData)
	if !ok {
		return email.EmailTemplate{}
	}
	return GeneratePaymentFailedEmail(paymentData, reason)
}

func (g *StandardTemplateGenerator) GeneratePaymentPendingEmail(payment interface{}) email.EmailTemplate {
	paymentData, ok := payment.(*PaymentData)
	if !ok {
		return email.EmailTemplate{}
	}
	return GeneratePaymentPendingEmail(paymentData)
}

func (g *StandardTemplateGenerator) GeneratePaymentRefundedEmail(payment interface{}) email.EmailTemplate {
	paymentData, ok := payment.(*PaymentData)
	if !ok {
		return email.EmailTemplate{}
	}
	return GeneratePaymentRefundedEmail(paymentData)
}
