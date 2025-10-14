package services

import (
	"fmt"
	"log"

	"ms-scheduling/internal/models"
)

// SendOrderConfirmationEmail sends order confirmation email
func (s *SubscriberService) SendOrderConfirmationEmail(subscriber *models.Subscriber, order *OrderCreatedEvent) error {
	log.Printf("Sending order confirmation email to %s for order %s", subscriber.SubscriberMail, order.OrderID)

	emailContent := s.generateOrderEmailTemplate(order)
	subject := fmt.Sprintf("Order Confirmation - %s", order.OrderID)

	return s.EmailService.SendEmail(subscriber.SubscriberMail, subject, emailContent)
}

func (s *SubscriberService) generateOrderEmailTemplate(order *OrderCreatedEvent) string {
	template := `
Dear Customer,

Your order has been confirmed!

Order Details:
- Order ID: %s
- Event ID: %s
- Session ID: %s
- Status: %s
- Total Price: $%.2f
- Created At: %s

Tickets:
%s

Thank you for your purchase!

Best regards,
Ticketly Team
`

	ticketDetails := ""
	for _, ticket := range order.Tickets {
		ticketDetails += fmt.Sprintf("- Seat: %s (%s) - $%.2f\n",
			ticket.SeatLabel, ticket.TierName, ticket.PriceAtPurchase)
	}

	return fmt.Sprintf(template,
		order.OrderID,
		order.EventID,
		order.SessionID,
		order.Status,
		order.Price,
		order.CreatedAt,
		ticketDetails,
	)
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
