package services

import (
	"database/sql"
	"fmt"
	"log"
	"ms-scheduling/internal/models"
)

type SubscriberService struct {
	DB             *sql.DB
	KeycloakClient *KeycloakClient
	EmailService   *EmailService
}

func NewSubscriberService(db *sql.DB, keycloakClient *KeycloakClient, emailService *EmailService) *SubscriberService {
	return &SubscriberService{
		DB:             db,
		KeycloakClient: keycloakClient,
		EmailService:   emailService,
	}
}

// GetOrCreateSubscriber gets subscriber by user ID or creates a new one
func (s *SubscriberService) GetOrCreateSubscriber(userID string) (*models.Subscriber, error) {
	// First, try to get subscriber from database by UserID
	subscriber, err := s.getSubscriberByUserID(userID)
	if err == nil {
		return subscriber, nil
	}

	// If not found, fetch email from Keycloak
	email, err := s.KeycloakClient.GetUserEmail(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user email from Keycloak: %v", err)
	}

	// Create new subscriber
	subscriber, err = s.createSubscriber(userID, email)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber: %v", err)
	}

	log.Printf("Created new subscriber for user %s with email %s", userID, email)
	return subscriber, nil
}

// getSubscriberByUserID retrieves subscriber from database using Keycloak UUID
func (s *SubscriberService) getSubscriberByUserID(userID string) (*models.Subscriber, error) {
	// Try to find by user_id (Keycloak UUID) first
	query := `
		SELECT subscriber_id, user_id, subscriber_mail, created_at 
		FROM subscribers 
		WHERE user_id = $1
	`

	var subscriber models.Subscriber
	err := s.DB.QueryRow(query, userID).Scan(
		&subscriber.SubscriberID,
		&subscriber.UserID,
		&subscriber.SubscriberMail,
		&subscriber.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &subscriber, nil
}

// createSubscriber creates a new subscriber in the database with both user_id and email
func (s *SubscriberService) createSubscriber(userID string, email string) (*models.Subscriber, error) {
	query := `
		INSERT INTO subscribers (user_id, subscriber_mail) 
		VALUES ($1, $2) 
		ON CONFLICT (subscriber_mail) DO UPDATE SET 
			user_id = EXCLUDED.user_id,
			created_at = subscribers.created_at
		RETURNING subscriber_id, user_id, subscriber_mail, created_at
	`

	var subscriber models.Subscriber
	err := s.DB.QueryRow(query, userID, email).Scan(
		&subscriber.SubscriberID,
		&subscriber.UserID,
		&subscriber.SubscriberMail,
		&subscriber.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &subscriber, nil
}

// AddSubscription adds a subscription for a subscriber
func (s *SubscriberService) AddSubscription(subscriberID int, category models.SubscriptionCategory, targetID int) error {
	query := `
		INSERT INTO subscriptions (subscriber_id, category, target_id) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (subscriber_id, category, target_id) DO NOTHING
	`

	_, err := s.DB.Exec(query, subscriberID, category, targetID)
	return err
}

// SendOrderConfirmationEmail sends order confirmation email
func (s *SubscriberService) SendOrderConfirmationEmail(subscriber *models.Subscriber, order *OrderCreatedEvent) error {
	log.Printf("Sending order confirmation email to %s for order %s", subscriber.SubscriberMail, order.OrderID)

	emailContent := s.generateOrderEmailTemplate(order)
	subject := fmt.Sprintf("Order Confirmation - %s", order.OrderID)

	return s.EmailService.SendEmail(subscriber.SubscriberMail, subject, emailContent)
}

// generateOrderEmailTemplate creates the email content
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
