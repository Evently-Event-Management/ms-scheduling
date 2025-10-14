package services

import (
	"log"

	"ms-scheduling/internal/models"
)

// SendOrderConfirmationEmail sends an order email based on order status
func (s *SubscriberService) SendOrderConfirmationEmail(subscriber *models.Subscriber, order *OrderCreatedEvent) error {
	log.Printf("Sending order email to %s for order %s with status %s", subscriber.SubscriberMail, order.OrderID, order.Status)

	// Determine email template type based on order status
	var emailType EmailType
	switch order.Status {
	case "completed":
		emailType = EmailOrderConfirmed
	case "pending":
		emailType = EmailOrderPending
	case "cancelled":
		emailType = EmailOrderCancelled
	case "processing":
		emailType = EmailOrderProcessing
	default:
		emailType = EmailOrderPending // Default to pending if status is unknown
	}

	// Generate email using our new template system
	emailTemplate := GenerateEmailTemplate(s.Config, emailType, order)

	// Send the email
	return s.EmailService.SendEmail(subscriber.SubscriberMail, emailTemplate.Subject, emailTemplate.HTML)
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
