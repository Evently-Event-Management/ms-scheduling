package services

import (
	"log"

	"ms-scheduling/internal/models"
)

// SendOrderConfirmationEmail sends an order email based on order status
func (s *SubscriberService) SendOrderConfirmationEmail(subscriber *models.Subscriber, order *OrderCreatedEvent) error {
	log.Printf("Sending order email to %s for order %s with status %s", subscriber.SubscriberMail, order.OrderID, order.Status)

	// Determine email template type based on order status
	var templateType EmailTemplateType
	switch order.Status {
	case "completed":
		templateType = OrderConfirmed
	case "pending":
		templateType = OrderPending
	case "cancelled":
		templateType = OrderCancelled
	case "processing":
		templateType = OrderProcessing
	default:
		templateType = OrderPending // Default to pending if status is unknown
	}

	// Generate HTML content
	emailContent := GenerateHTMLEmailTemplate(templateType, order)

	// Get email subject
	subject := GetEmailSubject(templateType, order.OrderID)

	// Send the email
	return s.EmailService.SendEmail(subscriber.SubscriberMail, subject, emailContent)
}

// OrderCreatedEvent represents the structure of the order.created Kafka event
type OrderCreatedEvent struct {
	OrderID        string   `json:"OrderID"`
	UserID         string   `json:"UserID"`
	EventID        string   `json:"EventID"`
	SessionID      string   `json:"SessionID"`
	OrganizationID string   `json:"OrganizationID"`
	Status         string   `json:"Status"`
	SubTotal       float64  `json:"SubTotal"`
	DiscountID     string   `json:"DiscountID"`
	DiscountCode   string   `json:"DiscountCode"`
	DiscountAmount float64  `json:"DiscountAmount"`
	Price          float64  `json:"Price"`
	CreatedAt      string   `json:"CreatedAt"`
	PaymentAT      string   `json:"PaymentAT"`
	Tickets        []Ticket `json:"tickets"`
}

type Ticket struct {
	TicketID        string  `json:"ticket_id"`
	OrderID         string  `json:"order_id"`
	SeatID          string  `json:"seat_id"`
	SeatLabel       string  `json:"seat_label"`
	Colour          string  `json:"colour"`
	TierID          string  `json:"tier_id"`
	TierName        string  `json:"tier_name"`
	PriceAtPurchase float64 `json:"price_at_purchase"`
	IssuedAt        string  `json:"issued_at"`
	CheckedIn       bool    `json:"checked_in"`
	CheckedInTime   string  `json:"checked_in_time"`
}
