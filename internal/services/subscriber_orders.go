package services

import (
	"log"

	"ms-scheduling/internal/email/templates"
	"ms-scheduling/internal/models"
)

// SendOrderConfirmationEmail sends an order email based on order status
func (s *SubscriberService) SendOrderConfirmationEmail(subscriber *models.Subscriber, order *OrderCreatedEvent) error {
	log.Printf("Sending order email to %s for order %s with status %s", subscriber.SubscriberMail, order.OrderID, order.Status)

	// Convert to OrderData format for new template system
	if s.EmailManager != nil {
		orderData := convertToOrderData(order)

		var err error
		switch order.Status {
		case "completed":
			err = s.EmailManager.SendOrderConfirmedEmail(subscriber.SubscriberMail, orderData)
		case "pending":
			err = s.EmailManager.SendOrderPendingEmail(subscriber.SubscriberMail, orderData)
		case "cancelled":
			err = s.EmailManager.SendOrderCancelledEmail(subscriber.SubscriberMail, orderData)
		case "processing":
			err = s.EmailManager.SendOrderUpdatedEmail(subscriber.SubscriberMail, orderData)
		default:
			err = s.EmailManager.SendOrderPendingEmail(subscriber.SubscriberMail, orderData)
		}

		return err
	}

	// Fallback to old template system
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
		emailType = EmailOrderPending
	}

	emailTemplate := GenerateEmailTemplate(s.Config, emailType, order)
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

// convertToOrderData converts OrderCreatedEvent to templates.OrderData
func convertToOrderData(order *OrderCreatedEvent) *templates.OrderData {
	// Convert tickets
	ticketData := make([]templates.TicketData, len(order.Tickets))
	for i, ticket := range order.Tickets {
		ticketData[i] = templates.TicketData{
			TicketID:        ticket.TicketID,
			SeatLabel:       ticket.SeatLabel,
			TierName:        ticket.TierName,
			PriceAtPurchase: ticket.PriceAtPurchase,
		}
	}

	return &templates.OrderData{
		OrderID:        order.OrderID,
		UserID:         order.UserID,
		EventID:        order.EventID,
		SessionID:      order.SessionID,
		OrganizationID: order.OrganizationID,
		Status:         order.Status,
		SubTotal:       order.SubTotal,
		DiscountCode:   order.DiscountCode,
		DiscountAmount: order.DiscountAmount,
		Price:          order.Price,
		CreatedAt:      order.CreatedAt,
		PaymentAt:      order.PaymentAT,
		Tickets:        ticketData,
		EventTitle:     "", // Will need to fetch from DB if needed
		SessionTitle:   "", // Will need to fetch from DB if needed
	}
}
