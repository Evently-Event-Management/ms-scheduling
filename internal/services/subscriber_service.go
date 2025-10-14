package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"ms-scheduling/internal/models"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SubscriberService struct {
	DB                *sql.DB
	KeycloakClient    *KeycloakClient
	EmailService      *EmailService
	HttpClient        *http.Client
	EventQueryService string
}

func NewSubscriberService(db *sql.DB, keycloakClient *KeycloakClient, emailService *EmailService) *SubscriberService {
	return &SubscriberService{
		DB:             db,
		KeycloakClient: keycloakClient,
		EmailService:   emailService,
		HttpClient:     &http.Client{Timeout: 10 * time.Second},
	}
}

// GetOrCreateSubscriber gets subscriber by user ID or creates a new one
func (s *SubscriberService) GetOrCreateSubscriber(userID string) (*models.Subscriber, error) {
	// First, try to get subscriber from database by UserID
	subscriber, err := s.getSubscriberByUserID(userID)
	if err == nil {
		return subscriber, nil
	}

	// If not found, try to fetch email from Keycloak
	email, err := s.KeycloakClient.GetUserEmail(userID)
	if err != nil {
		// Instead of failing, log the error and use a fallback email
		log.Printf("Warning: Failed to get user email from Keycloak: %v", err)
		// Use userID as part of a fallback email
		email = userID + "@example.com"
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
func (s *SubscriberService) AddSubscription(subscriberID int, category models.SubscriptionCategory, targetUUID string) error {
	query := `
		INSERT INTO subscriptions (subscriber_id, category, target_uuid) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (subscriber_id, category, target_uuid) DO NOTHING
	`

	_, err := s.DB.Exec(query, subscriberID, category, targetUUID)
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

// GetSessionSubscribers retrieves all subscribers for a specific session
func (s *SubscriberService) GetSessionSubscribers(sessionID string) ([]models.Subscriber, error) {
	query := `
		SELECT DISTINCT s.subscriber_id, s.subscriber_mail, s.user_id, s.created_at
		FROM subscribers s
		JOIN subscriptions sub ON s.subscriber_id = sub.subscriber_id
		WHERE sub.category = 'session' AND sub.target_uuid = $1`

	rows, err := s.DB.Query(query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("error querying session subscribers: %w", err)
	}
	defer rows.Close()

	var subscribers []models.Subscriber
	for rows.Next() {
		var subscriber models.Subscriber
		var userID sql.NullString

		err := rows.Scan(
			&subscriber.SubscriberID,
			&subscriber.SubscriberMail,
			&userID,
			&subscriber.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning subscriber: %w", err)
		}

		if userID.Valid {
			subscriber.UserID = &userID.String
		}

		subscribers = append(subscribers, subscriber)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating subscribers: %w", err)
	}

	return subscribers, nil
}

// ProcessSessionUpdate handles session update notifications from Debezium
func (s *SubscriberService) ProcessSessionUpdate(sessionUpdate *models.DebeziumSessionEvent) error {
	log.Printf("Processing session update event: %s", sessionUpdate.Payload.Operation)

	// Skip only initial snapshots
	if sessionUpdate.Payload.Operation == "r" {
		log.Printf("Skipping session update for initial snapshot operation: %s", sessionUpdate.Payload.Operation)
		return nil
	}

	var sessionID string

	// Get session ID from appropriate data based on operation
	if sessionUpdate.Payload.Operation == "d" {
		// For delete operations, get session ID from before data
		if sessionUpdate.Payload.Before != nil {
			sessionID = sessionUpdate.Payload.Before.ID
		} else {
			return fmt.Errorf("no before data available for session deletion")
		}
	} else {
		// For create/update operations, get session ID from after data
		if sessionUpdate.Payload.After != nil {
			sessionID = sessionUpdate.Payload.After.ID
		} else {
			return fmt.Errorf("no after data available for session update")
		}
	}

	// Get all subscribers for this session
	subscribers, err := s.GetSessionSubscribers(sessionID)
	if err != nil {
		return fmt.Errorf("error getting session subscribers: %w", err)
	}

	if len(subscribers) == 0 {
		log.Printf("No subscribers found for session ID: %s", sessionID)
		return nil
	}

	// Send notification emails
	return s.SendSessionUpdateEmails(subscribers, sessionUpdate)
}

// SendSessionUpdateEmails sends notification emails to all session subscribers
func (s *SubscriberService) SendSessionUpdateEmails(subscribers []models.Subscriber, sessionUpdate *models.DebeziumSessionEvent) error {
	log.Printf("Sending session update emails to %d subscribers", len(subscribers))

	for _, subscriber := range subscribers {
		subject, body := s.buildSessionUpdateEmail(subscriber, sessionUpdate)

		err := s.EmailService.SendEmail(subscriber.SubscriberMail, subject, body)
		if err != nil {
			log.Printf("Error sending session update email to %s: %v", subscriber.SubscriberMail, err)
			// Continue with other subscribers even if one fails
			continue
		}

		log.Printf("Session update email sent successfully to: %s", subscriber.SubscriberMail)
	}

	return nil
}

// ProcessEventUpdate handles event update notifications from Debezium
func (s *SubscriberService) ProcessEventUpdate(eventUpdate *models.DebeziumEventEvent) error {
	log.Printf("Processing event update event: %s", eventUpdate.Payload.Operation)

	// Skip only initial snapshots
	if eventUpdate.Payload.Operation == "r" {
		log.Printf("Skipping event update for initial snapshot operation: %s", eventUpdate.Payload.Operation)
		return nil
	}

	var eventID string

	// Get event ID from appropriate data based on operation
	if eventUpdate.Payload.Operation == "d" {
		// For delete operations, get event ID from before data
		if eventUpdate.Payload.Before != nil {
			eventID = eventUpdate.Payload.Before.ID
		} else {
			return fmt.Errorf("no before data available for event deletion")
		}
	} else {
		// For create/update operations, get event ID from after data
		if eventUpdate.Payload.After != nil {
			eventID = eventUpdate.Payload.After.ID
		} else {
			return fmt.Errorf("no after data available for event update")
		}
	}

	// Get all subscribers for this event
	subscribers, err := s.GetEventSubscribers(eventID)
	if err != nil {
		return fmt.Errorf("error getting event subscribers: %w", err)
	}

	if len(subscribers) == 0 {
		log.Printf("No subscribers found for event ID: %s", eventID)
		return nil
	}

	// Send notification emails
	return s.SendEventUpdateEmails(subscribers, eventUpdate)
}

// GetEventSubscribers retrieves all subscribers for a specific event
func (s *SubscriberService) GetEventSubscribers(eventID string) ([]models.Subscriber, error) {
	query := `
		SELECT DISTINCT s.subscriber_id, s.user_id, s.subscriber_mail, s.created_at 
		FROM subscribers s
		JOIN subscriptions sub ON s.subscriber_id = sub.subscriber_id
		WHERE sub.category = 'event' AND sub.target_uuid = $1
	`

	rows, err := s.DB.Query(query, eventID)
	if err != nil {
		return nil, fmt.Errorf("error querying event subscribers: %w", err)
	}
	defer rows.Close()

	var subscribers []models.Subscriber

	for rows.Next() {
		var subscriber models.Subscriber
		err := rows.Scan(
			&subscriber.SubscriberID,
			&subscriber.UserID,
			&subscriber.SubscriberMail,
			&subscriber.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning subscriber: %w", err)
		}
		subscribers = append(subscribers, subscriber)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event subscribers: %w", err)
	}

	return subscribers, nil
}

// SendEventUpdateEmails sends notification emails to all event subscribers
func (s *SubscriberService) SendEventUpdateEmails(subscribers []models.Subscriber, eventUpdate *models.DebeziumEventEvent) error {
	log.Printf("Sending event update emails to %d subscribers", len(subscribers))

	for _, subscriber := range subscribers {
		subject, body := s.buildEventUpdateEmail(subscriber, eventUpdate)

		err := s.EmailService.SendEmail(subscriber.SubscriberMail, subject, body)
		if err != nil {
			log.Printf("Error sending event update email to %s: %v", subscriber.SubscriberMail, err)
			// Continue with other subscribers even if one fails
			continue
		}

		log.Printf("Event update email sent successfully to: %s", subscriber.SubscriberMail)
	}

	return nil
}

// buildEventUpdateEmail creates the email content for event updates
func (s *SubscriberService) buildEventUpdateEmail(subscriber models.Subscriber, eventUpdate *models.DebeziumEventEvent) (string, string) {
	after := eventUpdate.Payload.After
	before := eventUpdate.Payload.Before
	operation := eventUpdate.Payload.Operation

	// Convert timestamp to readable format
	timestamp := time.UnixMilli(eventUpdate.Payload.Timestamp)

	var subject string
	var body strings.Builder

	// Handle different operations
	if operation == "d" {
		// Event deletion
		if before == nil {
			return "", ""
		}

		subject = fmt.Sprintf("Event Cancelled: %s", before.Title)

		body.WriteString("Dear Subscriber,\n\n")
		body.WriteString("⚠️ IMPORTANT: An event you're subscribed to has been CANCELLED/DELETED:\n\n")

		// Deleted event details
		body.WriteString("Cancelled Event Details:\n")
		body.WriteString(fmt.Sprintf("Event ID: %s\n", before.ID))
		body.WriteString(fmt.Sprintf("Title: %s\n", before.Title))
		body.WriteString(fmt.Sprintf("Description: %s\n", before.Description))
		body.WriteString(fmt.Sprintf("Status: %s\n", before.Status))
		body.WriteString(fmt.Sprintf("Created: %s\n", time.Unix(before.CreatedAt/1000000, 0).Format("2006-01-02 15:04:05")))
		body.WriteString(fmt.Sprintf("Cancelled: %s\n\n", timestamp.Format("2006-01-02 15:04:05")))

		body.WriteString("🔔 This event has been permanently removed from the schedule.\n")
		body.WriteString("📧 If you had tickets for sessions in this event, please check your email for refund information or contact support.\n\n")

	} else {
		// Event update or creation
		if after == nil {
			return "", ""
		}

		subject = fmt.Sprintf("Event Update: %s", after.Title)

		body.WriteString("Dear Subscriber,\n\n")
		body.WriteString("An event you're subscribed to has been updated:\n\n")

		// Event details
		body.WriteString(fmt.Sprintf("Event ID: %s\n", after.ID))
		body.WriteString(fmt.Sprintf("Title: %s\n", after.Title))
		body.WriteString(fmt.Sprintf("Description: %s\n", after.Description))
		body.WriteString(fmt.Sprintf("Status: %s\n", after.Status))
		body.WriteString(fmt.Sprintf("Created: %s\n", time.Unix(after.CreatedAt/1000000, 0).Format("2006-01-02 15:04:05")))
		body.WriteString(fmt.Sprintf("Updated: %s\n\n", timestamp.Format("2006-01-02 15:04:05")))

		// Show what changed
		if before != nil && operation == "u" {
			body.WriteString("Changes:\n")

			if before.Title != after.Title {
				body.WriteString(fmt.Sprintf("• Title: %s → %s\n", before.Title, after.Title))
			}

			if before.Description != after.Description {
				body.WriteString(fmt.Sprintf("• Description: %s → %s\n", before.Description, after.Description))
			}

			if before.Status != after.Status {
				body.WriteString(fmt.Sprintf("• Status: %s → %s\n", before.Status, after.Status))
			}

			if before.Overview != after.Overview {
				body.WriteString("• Overview: Updated\n")
			}

			if before.CategoryID != after.CategoryID {
				body.WriteString("• Category: Updated\n")
			}
		} else if operation == "c" {
			// New event notification
			body.WriteString("New Event Details:\n")
			body.WriteString(fmt.Sprintf("• Status: %s\n", after.Status))
			if after.Overview != "" {
				body.WriteString(fmt.Sprintf("• Overview: %s\n", after.Overview))
			}
		}

		// Special handling for status changes
		if operation == "u" && before != nil && before.Status != after.Status {
			body.WriteString("\n🔔 Status Change Notification:\n")
			switch after.Status {
			case "APPROVED":
				body.WriteString("✅ This event has been APPROVED and is now available for booking!\n")
			case "REJECTED":
				body.WriteString("❌ This event has been REJECTED.")
				if after.RejectionReason != "" {
					body.WriteString(fmt.Sprintf(" Reason: %s", after.RejectionReason))
				}
				body.WriteString("\n")
			case "PENDING":
				body.WriteString("⏳ This event is now under review.\n")
			}
		}
	}

	body.WriteString("\nBest regards,\nTicketly Team")

	return subject, body.String()
}

// buildSessionUpdateEmail creates the email content for session updates
func (s *SubscriberService) buildSessionUpdateEmail(subscriber models.Subscriber, sessionUpdate *models.DebeziumSessionEvent) (string, string) {
	after := sessionUpdate.Payload.After
	before := sessionUpdate.Payload.Before
	operation := sessionUpdate.Payload.Operation

	// Convert timestamp to readable format
	timestamp := time.UnixMilli(sessionUpdate.Payload.Timestamp)

	var subject string
	var body strings.Builder

	// Handle different operations
	if operation == "d" {
		// Session deletion
		if before == nil {
			return "", ""
		}

		subject = fmt.Sprintf("Session Cancelled: Session %s", before.ID)

		body.WriteString("Dear Subscriber,\n\n")
		body.WriteString("⚠️ IMPORTANT: A session you're subscribed to has been CANCELLED/DELETED:\n\n")

		// Deleted session details
		body.WriteString("Cancelled Session Details:\n")
		body.WriteString(fmt.Sprintf("Session ID: %s\n", before.ID))
		body.WriteString(fmt.Sprintf("Event ID: %s\n", before.EventID))
		body.WriteString(fmt.Sprintf("Status: %s\n", before.Status))
		body.WriteString(fmt.Sprintf("Session Type: %s\n", before.SessionType))
		body.WriteString(fmt.Sprintf("Start Time: %s\n", time.Unix(before.StartTime/1000000, 0).Format("2006-01-02 15:04:05")))
		body.WriteString(fmt.Sprintf("End Time: %s\n", time.Unix(before.EndTime/1000000, 0).Format("2006-01-02 15:04:05")))
		body.WriteString(fmt.Sprintf("Cancelled: %s\n\n", timestamp.Format("2006-01-02 15:04:05")))

		// Parse venue details if available
		if before.VenueDetails != "" {
			body.WriteString("Venue Information:\n")
			var venueMap map[string]interface{}
			if err := json.Unmarshal([]byte(before.VenueDetails), &venueMap); err == nil {
				if name, ok := venueMap["name"].(string); ok {
					body.WriteString(fmt.Sprintf("Venue: %s\n", name))
				}
				if address, ok := venueMap["address"].(string); ok {
					body.WriteString(fmt.Sprintf("Address: %s\n", address))
				}
			}
			body.WriteString("\n")
		}

		body.WriteString("🔔 This session has been permanently removed from the schedule.\n")
		body.WriteString("📧 If you had tickets for this session, please check your email for refund information or contact support.\n\n")

	} else {
		// Session update or creation
		if after == nil {
			return "", ""
		}

		subject = fmt.Sprintf("Session Update: Session %s", after.ID)

		body.WriteString("Dear Subscriber,\n\n")
		body.WriteString("A session you're subscribed to has been updated:\n\n")

		// Session details
		body.WriteString(fmt.Sprintf("Session ID: %s\n", after.ID))
		body.WriteString(fmt.Sprintf("Event ID: %s\n", after.EventID))
		body.WriteString(fmt.Sprintf("Status: %s\n", after.Status))
		body.WriteString(fmt.Sprintf("Session Type: %s\n", after.SessionType))
		body.WriteString(fmt.Sprintf("Start Time: %s\n", time.Unix(after.StartTime/1000000, 0).Format("2006-01-02 15:04:05")))
		body.WriteString(fmt.Sprintf("End Time: %s\n", time.Unix(after.EndTime/1000000, 0).Format("2006-01-02 15:04:05")))
		body.WriteString(fmt.Sprintf("Updated: %s\n\n", timestamp.Format("2006-01-02 15:04:05")))

		// Show what changed
		if before != nil && operation == "u" {
			body.WriteString("Changes:\n")

			if before.Status != after.Status {
				body.WriteString(fmt.Sprintf("• Status: %s → %s\n", before.Status, after.Status))
			}

			if before.StartTime != after.StartTime {
				beforeTime := time.Unix(before.StartTime/1000000, 0).Format("2006-01-02 15:04:05")
				afterTime := time.Unix(after.StartTime/1000000, 0).Format("2006-01-02 15:04:05")
				body.WriteString(fmt.Sprintf("• Start Time: %s → %s\n", beforeTime, afterTime))
			}

			if before.EndTime != after.EndTime {
				beforeTime := time.Unix(before.EndTime/1000000, 0).Format("2006-01-02 15:04:05")
				afterTime := time.Unix(after.EndTime/1000000, 0).Format("2006-01-02 15:04:05")
				body.WriteString(fmt.Sprintf("• End Time: %s → %s\n", beforeTime, afterTime))
			}

			if before.SessionType != after.SessionType {
				body.WriteString(fmt.Sprintf("• Session Type: %s → %s\n", before.SessionType, after.SessionType))
			}

			if before.VenueDetails != after.VenueDetails {
				body.WriteString("• Venue Details: Updated\n")
			}

			if before.SalesStartTime != after.SalesStartTime {
				var beforeSales, afterSales string
				if before.SalesStartTime > 0 {
					beforeSales = time.Unix(before.SalesStartTime/1000000, 0).Format("2006-01-02 15:04:05")
				} else {
					beforeSales = "Not set"
				}
				if after.SalesStartTime > 0 {
					afterSales = time.Unix(after.SalesStartTime/1000000, 0).Format("2006-01-02 15:04:05")
				} else {
					afterSales = "Not set"
				}
				body.WriteString(fmt.Sprintf("• Sales Start Time: %s → %s\n", beforeSales, afterSales))
			}
		} else if operation == "c" {
			// New session notification
			body.WriteString("New Session Details:\n")
			body.WriteString(fmt.Sprintf("• Status: %s\n", after.Status))
			body.WriteString(fmt.Sprintf("• Session Type: %s\n", after.SessionType))
			if after.SalesStartTime > 0 {
				salesTime := time.Unix(after.SalesStartTime/1000000, 0).Format("2006-01-02 15:04:05")
				body.WriteString(fmt.Sprintf("• Sales Start Time: %s\n", salesTime))
			}
		}
	}

	body.WriteString("\nBest regards,\nTicketly Team")

	return subject, body.String()
}

// GetOrganizationSubscribers retrieves all subscribers for a specific organization
func (s *SubscriberService) GetOrganizationSubscribers(organizationID string) ([]models.Subscriber, error) {
	// Query subscribers who have subscribed to the organization
	query := `
		SELECT DISTINCT s.subscriber_id, s.user_id, s.subscriber_mail, s.created_at 
		FROM subscribers s
		JOIN subscriptions sub ON s.subscriber_id = sub.subscriber_id
		WHERE sub.category = 'organization' AND sub.target_uuid = $1
	`

	rows, err := s.DB.Query(query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("error querying organization subscribers: %w", err)
	}
	defer rows.Close()

	var subscribers []models.Subscriber

	for rows.Next() {
		var subscriber models.Subscriber
		err := rows.Scan(
			&subscriber.SubscriberID,
			&subscriber.UserID,
			&subscriber.SubscriberMail,
			&subscriber.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning organization subscriber: %w", err)
		}
		subscribers = append(subscribers, subscriber)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating organization subscribers: %w", err)
	}

	return subscribers, nil
}

// ProcessEventCreation handles event creation notifications from Debezium
func (s *SubscriberService) ProcessEventCreation(eventUpdate *models.DebeziumEventEvent) error {
	log.Printf("Processing event creation notification: %s", eventUpdate.Payload.Operation)

	// Only handle creation operations
	if eventUpdate.Payload.Operation != "c" {
		return nil
	}

	// Get organization ID from after data (for creation operations)
	if eventUpdate.Payload.After == nil {
		return fmt.Errorf("no after data available for event creation")
	}

	organizationID := eventUpdate.Payload.After.OrganizationID
	eventID := eventUpdate.Payload.After.ID

	log.Printf("Processing event creation for event %s in organization %s", eventID, organizationID)

	// Get all subscribers for this organization
	subscribers, err := s.GetOrganizationSubscribers(organizationID)
	if err != nil {
		return fmt.Errorf("error getting organization subscribers: %w", err)
	}

	if len(subscribers) == 0 {
		log.Printf("No organization subscribers found for organization ID: %s", organizationID)
		return nil
	}

	log.Printf("Found %d subscribers for organization %s", len(subscribers), organizationID)

	// Send notification emails
	return s.SendEventCreationEmails(subscribers, eventUpdate)
}

// SendEventCreationEmails sends notification emails to all organization subscribers for new events
func (s *SubscriberService) SendEventCreationEmails(subscribers []models.Subscriber, eventUpdate *models.DebeziumEventEvent) error {
	log.Printf("Sending event creation emails to %d subscribers", len(subscribers))

	for _, subscriber := range subscribers {
		subject, body := s.buildEventCreationEmail(subscriber, eventUpdate)

		err := s.EmailService.SendEmail(subscriber.SubscriberMail, subject, body)
		if err != nil {
			log.Printf("Error sending event creation email to %s: %v", subscriber.SubscriberMail, err)
			// Continue with other subscribers even if one fails
			continue
		}

		log.Printf("Event creation email sent successfully to: %s", subscriber.SubscriberMail)
	}

	return nil
}

// buildEventCreationEmail creates the email content for new event notifications
func (s *SubscriberService) buildEventCreationEmail(subscriber models.Subscriber, eventUpdate *models.DebeziumEventEvent) (string, string) {
	after := eventUpdate.Payload.After

	// Convert timestamp to readable format
	timestamp := time.UnixMilli(eventUpdate.Payload.Timestamp)
	createdAt := models.MicroTimestampToTime(after.CreatedAt)

	subject := fmt.Sprintf("🎉 New Event Created: %s", after.Title)

	var body strings.Builder
	body.WriteString(fmt.Sprintf("Hello %s,\n\n", subscriber.SubscriberMail))
	body.WriteString("🎉 A new event has been created in your subscribed organization!\n\n")

	body.WriteString("Event Details:\n")
	body.WriteString(fmt.Sprintf("• Title: %s\n", after.Title))
	body.WriteString(fmt.Sprintf("• Status: %s\n", after.Status))

	if after.Description != "" {
		body.WriteString(fmt.Sprintf("• Description: %s\n", after.Description))
	}

	if after.Overview != "" {
		body.WriteString(fmt.Sprintf("• Overview: %s\n", after.Overview))
	}

	body.WriteString(fmt.Sprintf("• Created: %s\n", createdAt.Format("2006-01-02 15:04:05")))
	body.WriteString(fmt.Sprintf("• Event ID: %s\n", after.ID))
	body.WriteString(fmt.Sprintf("• Organization ID: %s\n", after.OrganizationID))

	if after.CategoryID != "" {
		body.WriteString(fmt.Sprintf("• Category ID: %s\n", after.CategoryID))
	}

	body.WriteString(fmt.Sprintf("\n📅 Notification sent at: %s\n", timestamp.Format("2006-01-02 15:04:05")))

	if after.Status == "PENDING" {
		body.WriteString("\n⏳ This event is currently pending approval. You'll be notified when it's approved and ready for booking.\n")
	} else if after.Status == "APPROVED" {
		body.WriteString("\n✅ This event is approved and ready for booking!\n")
	}

	body.WriteString("\nStay tuned for more updates about this event!")
	body.WriteString("\n\nBest regards,\nTicketly Team")

	return subject, body.String()
}

// ProcessSessionReminder handles generic session reminder email notifications
// This is the legacy method that can handle any type of reminder
func (s *SubscriberService) ProcessSessionReminder(sessionID string) error {
	log.Printf("Processing generic session reminder email for session ID: %s", sessionID)

	// Get subscribers and session details
	allSubscribers, sessionDetails, err := s.getSubscribersAndSessionDetails(sessionID)
	if err != nil {
		return err
	}

	if len(allSubscribers) == 0 {
		log.Printf("No subscribers found for session %s reminder", sessionID)
		return nil
	}

	// Send reminder emails
	return s.SendSessionReminderEmails(allSubscribers, sessionDetails)
}

// ProcessSessionStartReminder handles session start reminder email notifications (1 day before session)
func (s *SubscriberService) ProcessSessionStartReminder(sessionID string) error {
	log.Printf("Processing session START reminder email for session ID: %s (1 day before)", sessionID)

	// Get subscribers and session details
	allSubscribers, sessionDetails, err := s.getSubscribersAndSessionDetails(sessionID)
	if err != nil {
		return err
	}

	if len(allSubscribers) == 0 {
		log.Printf("No subscribers found for session %s start reminder", sessionID)
		return nil
	}

	// Send session start reminder emails with specific template
	return s.SendSessionStartReminderEmails(allSubscribers, sessionDetails)
}

// ProcessSessionSaleReminder handles session on-sale reminder email notifications (30 min before sales start)
func (s *SubscriberService) ProcessSessionSaleReminder(sessionID string) error {
	log.Printf("Processing session ON-SALE reminder email for session ID: %s", sessionID)

	// Get subscribers and session details
	allSubscribers, sessionDetails, err := s.getSubscribersAndSessionDetails(sessionID)
	if err != nil {
		return err
	}

	if len(allSubscribers) == 0 {
		log.Printf("No subscribers found for session %s sales reminder", sessionID)
		return nil
	}

	// Send sales start reminder emails with specific template
	return s.SendSessionSalesReminderEmails(allSubscribers, sessionDetails)
}

// Helper function to avoid code duplication in the reminder processors
func (s *SubscriberService) getSubscribersAndSessionDetails(sessionID string) ([]models.Subscriber, *SessionReminderInfo, error) {
	// Get all subscribers for this session
	sessionSubscribers, err := s.GetSessionSubscribers(sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting session subscribers: %w", err)
	}

	// Fetch session details first - this will contain the event ID
	sessionDetails, err := s.getSessionDetails(sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting session details: %w", err)
	}

	// Now that we have the eventID from session details, get event subscribers
	var eventSubscribers []models.Subscriber
	if sessionDetails.EventID != "" {
		eventSubscribers, err = s.GetEventSubscribers(sessionDetails.EventID)
		if err != nil {
			log.Printf("Warning: Could not get event subscribers for event %s: %v", sessionDetails.EventID, err)
			// Continue with just session subscribers
		}
	}

	// Combine and deduplicate subscribers
	allSubscribers := s.combineAndDeduplicateSubscribers(sessionSubscribers, eventSubscribers)

	return allSubscribers, sessionDetails, nil
}

// getEventIDFromSession retrieves the event ID associated with a session using the Event Query API
func (s *SubscriberService) getEventIDFromSession(sessionID string) (string, error) {
	if s.EventQueryService == "" {
		return "", fmt.Errorf("event query service URL not configured")
	}

	// Use the extended session info API to get session details including the event ID
	sessionInfo, err := s.getSessionDetailsFromAPI(sessionID)
	if err != nil {
		return "", err
	}

	return sessionInfo.EventID, nil
}

// getSessionDetailsFromAPI fetches session details from the Event Query API
func (s *SubscriberService) getSessionDetailsFromAPI(sessionID string) (*models.SessionExtendedInfo, error) {
	if s.EventQueryService == "" {
		return nil, fmt.Errorf("event query service URL not configured")
	}

	if s.HttpClient == nil {
		s.HttpClient = &http.Client{Timeout: 10 * time.Second}
	}

	// Create the API URL for fetching extended session info
	apiURL := fmt.Sprintf("%s/v1/events/sessions/%s/extended-info", s.EventQueryService, sessionID)
	log.Printf("Fetching session details from: %s", apiURL)

	// Make the API request
	resp, err := s.HttpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch session info: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned non-OK status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response
	var sessionInfo models.SessionExtendedInfo
	if err := json.NewDecoder(resp.Body).Decode(&sessionInfo); err != nil {
		return nil, fmt.Errorf("failed to decode session info: %w", err)
	}

	return &sessionInfo, nil
}

// getEventDetailsFromAPI fetches event details from the Event Query API
func (s *SubscriberService) getEventDetailsFromAPI(eventID string) (*models.EventBasicInfo, error) {
	if s.EventQueryService == "" {
		return nil, fmt.Errorf("event query service URL not configured")
	}

	if s.HttpClient == nil {
		s.HttpClient = &http.Client{Timeout: 10 * time.Second}
	}

	// Create the API URL for fetching basic event info
	apiURL := fmt.Sprintf("%s/v1/events/%s/basic-info", s.EventQueryService, eventID)
	log.Printf("Fetching event details from: %s", apiURL)

	// Make the API request
	resp, err := s.HttpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch event info: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned non-OK status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response
	var eventInfo models.EventBasicInfo
	if err := json.NewDecoder(resp.Body).Decode(&eventInfo); err != nil {
		return nil, fmt.Errorf("failed to decode event info: %w", err)
	}

	return &eventInfo, nil
}

// getSessionDetails retrieves session information for reminder emails using the API
func (s *SubscriberService) getSessionDetails(sessionID string) (*SessionReminderInfo, error) {
	// Get session details from the API
	sessionData, err := s.getSessionDetailsFromAPI(sessionID)
	if err != nil {
		return nil, err
	}

	// Convert API model to our internal model
	session := &SessionReminderInfo{
		SessionID:      sessionData.SessionID,
		EventID:        sessionData.EventID,
		EventTitle:     sessionData.EventTitle,
		StartTime:      sessionData.StartTime.UnixMicro(),
		EndTime:        sessionData.EndTime.UnixMicro(),
		Status:         sessionData.Status,
		SessionType:    sessionData.SessionType,
		SalesStartTime: sessionData.SalesStartTime.UnixMicro(),
	}

	// Handle venue details
	venueDetails, err := json.Marshal(sessionData.VenueDetails)
	if err == nil {
		session.VenueDetails = string(venueDetails)
	}

	// If the event title is not included in the session details, try to fetch it separately
	if session.EventTitle == "" && session.EventID != "" {
		eventData, err := s.getEventDetailsFromAPI(session.EventID)
		if err == nil {
			session.EventTitle = eventData.Title
		} else {
			log.Printf("Could not get event title for event %s: %v", session.EventID, err)
		}
	}

	return session, nil
}

// combineAndDeduplicateSubscribers merges two subscriber lists and removes duplicates
func (s *SubscriberService) combineAndDeduplicateSubscribers(sessionSubs, eventSubs []models.Subscriber) []models.Subscriber {
	subscriberMap := make(map[int]models.Subscriber)

	// Add session subscribers
	for _, sub := range sessionSubs {
		subscriberMap[sub.SubscriberID] = sub
	}

	// Add event subscribers (will overwrite duplicates, which is fine)
	for _, sub := range eventSubs {
		subscriberMap[sub.SubscriberID] = sub
	}

	// Convert map back to slice
	var result []models.Subscriber
	for _, sub := range subscriberMap {
		result = append(result, sub)
	}

	return result
}

// SendSessionReminderEmails sends generic reminder emails to all subscribers
func (s *SubscriberService) SendSessionReminderEmails(subscribers []models.Subscriber, sessionInfo *SessionReminderInfo) error {
	log.Printf("Sending generic session reminder emails to %d subscribers", len(subscribers))

	for _, subscriber := range subscribers {
		subject, body := s.buildSessionReminderEmail(subscriber, sessionInfo)

		err := s.EmailService.SendEmail(subscriber.SubscriberMail, subject, body)
		if err != nil {
			log.Printf("Error sending session reminder email to %s: %v", subscriber.SubscriberMail, err)
			// Continue with other subscribers even if one fails
			continue
		}

		log.Printf("Session reminder email sent successfully to: %s", subscriber.SubscriberMail)
	}

	return nil
}

// SendSessionStartReminderEmails sends session start reminder emails (1 day before)
func (s *SubscriberService) SendSessionStartReminderEmails(subscribers []models.Subscriber, sessionInfo *SessionReminderInfo) error {
	log.Printf("Sending session START reminder emails to %d subscribers (1 day before)", len(subscribers))

	for _, subscriber := range subscribers {
		subject, body := s.buildSessionStartReminderEmail(subscriber, sessionInfo)

		err := s.EmailService.SendEmail(subscriber.SubscriberMail, subject, body)
		if err != nil {
			log.Printf("Error sending session start reminder email to %s: %v", subscriber.SubscriberMail, err)
			// Continue with other subscribers even if one fails
			continue
		}

		log.Printf("Session start reminder email sent successfully to: %s", subscriber.SubscriberMail)
	}

	return nil
}

// SendSessionSalesReminderEmails sends sales start reminder emails (30 min before)
func (s *SubscriberService) SendSessionSalesReminderEmails(subscribers []models.Subscriber, sessionInfo *SessionReminderInfo) error {
	log.Printf("Sending session SALES reminder emails to %d subscribers", len(subscribers))

	for _, subscriber := range subscribers {
		subject, body := s.buildSessionSalesReminderEmail(subscriber, sessionInfo)

		err := s.EmailService.SendEmail(subscriber.SubscriberMail, subject, body)
		if err != nil {
			log.Printf("Error sending sales start reminder email to %s: %v", subscriber.SubscriberMail, err)
			// Continue with other subscribers even if one fails
			continue
		}

		log.Printf("Sales start reminder email sent successfully to: %s", subscriber.SubscriberMail)
	}

	return nil
}

// buildSessionReminderEmail creates the email content for session reminders
func (s *SubscriberService) buildSessionReminderEmail(subscriber models.Subscriber, sessionInfo *SessionReminderInfo) (string, string) {
	// Convert timestamps to readable format
	startTime := models.MicroTimestampToTime(sessionInfo.StartTime)
	endTime := models.MicroTimestampToTime(sessionInfo.EndTime)

	// Get subscriber name if possible
	subscriberName := ""
	if subscriber.UserID != nil && *subscriber.UserID != "" {
		// Try to get user details from Keycloak
		userDetails, err := s.KeycloakClient.GetUserDetails(*subscriber.UserID)
		if err == nil && userDetails != nil {
			if userDetails.FirstName != "" && userDetails.LastName != "" {
				subscriberName = fmt.Sprintf("%s %s", userDetails.FirstName, userDetails.LastName)
			} else if userDetails.FirstName != "" {
				subscriberName = userDetails.FirstName
			}
		} else {
			log.Printf("Failed to get Keycloak user details: %v", err)
		}
	}

	// Use email as fallback if name not available
	if subscriberName == "" {
		// Extract name from email if possible
		emailParts := strings.Split(subscriber.SubscriberMail, "@")
		subscriberName = emailParts[0]
	}

	var eventTitle string
	if sessionInfo.EventTitle != "" {
		eventTitle = sessionInfo.EventTitle
	} else {
		eventTitle = "Your Event"
	}

	subject := fmt.Sprintf("🔔 Reminder: %s is tomorrow!", eventTitle)

	// Calculate session duration
	duration := endTime.Sub(startTime)
	durationHours := int(duration.Hours())
	durationMinutes := int(duration.Minutes()) % 60

	// Format duration string
	var durationStr string
	if durationHours > 0 {
		if durationMinutes > 0 {
			durationStr = fmt.Sprintf("%d hours %d minutes", durationHours, durationMinutes)
		} else {
			durationStr = fmt.Sprintf("%d hours", durationHours)
		}
	} else {
		durationStr = fmt.Sprintf("%d minutes", durationMinutes)
	}

	// Format date and time more user-friendly
	dateStr := startTime.Format("Monday, January 2, 2006")
	startTimeStr := startTime.Format("3:04 PM")
	endTimeStr := endTime.Format("3:04 PM")

	// Generate calendar links
	calendarMsg := "\n<p><strong>📱 Add to Calendar:</strong> "
	calendarMsg += fmt.Sprintf("<a href=\"https://calendar.google.com/calendar/render?action=TEMPLATE&text=%s&dates=%s/%s&details=%s at %s&location=%s\">Google Calendar</a> | ",
		url.QueryEscape(eventTitle),
		startTime.Format("20060102T150405"),
		endTime.Format("20060102T150405"),
		url.QueryEscape(eventTitle),
		url.QueryEscape(sessionInfo.VenueDetails),
		url.QueryEscape(sessionInfo.VenueDetails))
	calendarMsg += fmt.Sprintf("<a href=\"webcal://ticketly.com/calendar/event-%s.ics\">Apple Calendar</a></p>", sessionInfo.SessionID)

	// Build HTML email body
	var body strings.Builder
	body.WriteString(fmt.Sprintf("<h2>Hello %s!</h2>", subscriberName))
	body.WriteString("<p><strong>🔔 This is a friendly reminder that you have a session starting tomorrow!</strong></p>")

	body.WriteString("<div style=\"background-color: #f8f9fa; padding: 15px; border-radius: 5px; margin: 20px 0;\">")
	body.WriteString("<h3 style=\"color: #007bff; margin-top: 0;\">Session Details</h3>")

	// Event info section
	body.WriteString("<div style=\"margin-bottom: 20px;\">")
	if sessionInfo.EventTitle != "" {
		body.WriteString(fmt.Sprintf("<h4 style=\"margin-bottom: 5px;\">%s</h4>", sessionInfo.EventTitle))
	}
	body.WriteString(fmt.Sprintf("<p><strong>Type:</strong> %s</p>", sessionInfo.SessionType))
	body.WriteString(fmt.Sprintf("<p><strong>Date:</strong> %s</p>", dateStr))
	body.WriteString(fmt.Sprintf("<p><strong>Time:</strong> %s - %s (%s)</p>", startTimeStr, endTimeStr, durationStr))

	// Add venue details if available
	if sessionInfo.VenueDetails != "" {
		body.WriteString(fmt.Sprintf("<p><strong>Location:</strong> %s</p>", sessionInfo.VenueDetails))
	}

	// Status-specific messaging
	if sessionInfo.Status == "ON_SALE" {
		body.WriteString("<p><span style=\"color: #28a745; font-weight: bold;\">🎫 TICKETS ON SALE NOW</span> - Don't forget to purchase your tickets!</p>")
	} else if sessionInfo.Status == "SOLD_OUT" {
		body.WriteString("<p><span style=\"color: #dc3545; font-weight: bold;\">SOLD OUT</span> - This session is sold out.</p>")
	} else if sessionInfo.Status == "PENDING" {
		body.WriteString("<p><span style=\"color: #ffc107; font-weight: bold;\">⏳ PENDING CONFIRMATION</span> - We'll update you if there are any changes.</p>")
	} else if sessionInfo.Status == "CONFIRMED" {
		body.WriteString("<p><span style=\"color: #28a745; font-weight: bold;\">✅ CONFIRMED</span> - This session is confirmed to take place as scheduled.</p>")
	}
	body.WriteString("</div>")

	// Session ID for reference
	body.WriteString(fmt.Sprintf("<p style=\"font-size: 12px; color: #6c757d;\">Reference #: %s</p>", sessionInfo.SessionID))
	body.WriteString("</div>")

	// Add countdown and calendar links
	body.WriteString("<p style=\"font-size: 18px; font-weight: bold; color: #007bff;\">⏰ This session starts in approximately 24 hours!</p>")
	body.WriteString(calendarMsg)

	// Add checklist and recommendations
	body.WriteString("<div style=\"background-color: #e9ecef; padding: 15px; border-radius: 5px; margin: 20px 0;\">")
	body.WriteString("<h4>📋 Pre-Session Checklist:</h4>")
	body.WriteString("<ul>")
	body.WriteString("<li>Set a reminder on your phone</li>")
	body.WriteString("<li>Check the venue location and plan your route</li>")
	body.WriteString("<li>Prepare any required documents or tickets</li>")
	body.WriteString("<li>Plan your travel time with extra buffer</li>")
	body.WriteString("</ul>")
	body.WriteString("</div>")

	body.WriteString("<p>We're excited to see you tomorrow! 🎉</p>")
	body.WriteString("<p>Best regards,<br>The Ticketly Team</p>")

	// Unsubscribe option
	body.WriteString("<p style=\"font-size: 12px; color: #6c757d; margin-top: 30px;\">")
	body.WriteString(fmt.Sprintf("To unsubscribe from these notifications, <a href=\"https://ticketly.com/unsubscribe/%s\">click here</a>.", sessionInfo.SessionID))
	body.WriteString("</p>")

	return subject, body.String()
}

// SessionReminderInfo holds session information for reminder emails
type SessionReminderInfo struct {
	SessionID      string
	EventID        string
	EventTitle     string
	StartTime      int64
	EndTime        int64
	Status         string
	VenueDetails   string
	SessionType    string
	SalesStartTime int64
}
