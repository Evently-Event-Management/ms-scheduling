package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"ms-scheduling/internal/models"
)

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

func (s *SubscriberService) ProcessSessionUpdate(sessionUpdate *models.DebeziumSessionEvent) error {
	log.Printf("Processing session update event: %s", sessionUpdate.Payload.Operation)

	if sessionUpdate.Payload.Operation == "r" {
		log.Printf("Skipping session update for initial snapshot operation: %s", sessionUpdate.Payload.Operation)
		return nil
	}

	var sessionID string
	if sessionUpdate.Payload.Operation == "d" {
		if sessionUpdate.Payload.Before != nil {
			sessionID = sessionUpdate.Payload.Before.ID
		} else {
			return fmt.Errorf("no before data available for session deletion")
		}
	} else {
		if sessionUpdate.Payload.After != nil {
			sessionID = sessionUpdate.Payload.After.ID
		} else {
			return fmt.Errorf("no after data available for session update")
		}
	}

	eventID := ""
	if sessionUpdate.Payload.After != nil {
		eventID = sessionUpdate.Payload.After.EventID
	} else if sessionUpdate.Payload.Before != nil {
		eventID = sessionUpdate.Payload.Before.EventID
	}

	subscribers, err := s.GetSessionSubscribers(sessionID)
	if err != nil {
		return fmt.Errorf("error getting session subscribers: %w", err)
	}

	// Notify event subscribers about new session creations
	if sessionUpdate.Payload.Operation == "c" && eventID != "" {
		eventSubscribers, err := s.GetEventSubscribers(eventID)
		if err != nil {
			log.Printf("Error getting event subscribers for new session notification: %v", err)
		} else if len(eventSubscribers) > 0 {
			if err := s.SendSessionCreationEmails(eventSubscribers, sessionUpdate); err != nil {
				log.Printf("Error sending session creation emails: %v", err)
			}
		}
	}

	if len(subscribers) == 0 {
		log.Printf("No subscribers found for session ID: %s", sessionID)
		return nil
	}

	return s.SendSessionUpdateEmails(subscribers, sessionUpdate)
}

func (s *SubscriberService) SendSessionUpdateEmails(subscribers []models.Subscriber, sessionUpdate *models.DebeziumSessionEvent) error {
	log.Printf("Sending session update emails to %d subscribers", len(subscribers))

	for _, subscriber := range subscribers {
		subject, body := s.buildSessionUpdateEmail(subscriber, sessionUpdate)

		err := s.EmailService.SendEmail(subscriber.SubscriberMail, subject, body)
		if err != nil {
			log.Printf("Error sending session update email to %s: %v", subscriber.SubscriberMail, err)
			continue
		}

		log.Printf("Session update email sent successfully to: %s", subscriber.SubscriberMail)
	}

	return nil
}

func (s *SubscriberService) SendSessionCreationEmails(subscribers []models.Subscriber, sessionUpdate *models.DebeziumSessionEvent) error {
	log.Printf("Sending session creation emails to %d event subscribers", len(subscribers))

	for _, subscriber := range subscribers {
		subject, body := s.buildSessionCreationEmail(subscriber, sessionUpdate)
		if subject == "" {
			continue
		}

		if err := s.EmailService.SendEmail(subscriber.SubscriberMail, subject, body); err != nil {
			log.Printf("Error sending session creation email to %s: %v", subscriber.SubscriberMail, err)
			continue
		}

		log.Printf("Session creation email sent successfully to: %s", subscriber.SubscriberMail)
	}

	return nil
}

func (s *SubscriberService) buildSessionCreationEmail(subscriber models.Subscriber, sessionUpdate *models.DebeziumSessionEvent) (string, string) {
	after := sessionUpdate.Payload.After
	if after == nil {
		return "", ""
	}

	subject := fmt.Sprintf("New Session Available for Event %s", after.EventID)

	start := time.Unix(after.StartTime/1000000, 0)
	end := time.Unix(after.EndTime/1000000, 0)

	var body strings.Builder
	body.WriteString("Hello,\n\n")
	body.WriteString("A new session has been added to an event you're following:\n\n")
	body.WriteString(fmt.Sprintf("Session ID: %s\n", after.ID))
	body.WriteString(fmt.Sprintf("Event ID: %s\n", after.EventID))
	body.WriteString(fmt.Sprintf("Status: %s\n", after.Status))
	body.WriteString(fmt.Sprintf("Session Type: %s\n", after.SessionType))
	body.WriteString(fmt.Sprintf("Start Time: %s\n", start.Format("2006-01-02 15:04:05")))
	body.WriteString(fmt.Sprintf("End Time: %s\n\n", end.Format("2006-01-02 15:04:05")))

	if after.VenueDetails != "" {
		body.WriteString("Venue Details:\n")
		var venue map[string]interface{}
		if err := json.Unmarshal([]byte(after.VenueDetails), &venue); err == nil {
			for key, value := range venue {
				body.WriteString(fmt.Sprintf("• %s: %v\n", strings.Title(key), value))
			}
		} else {
			body.WriteString(after.VenueDetails + "\n")
		}
		body.WriteString("\n")
	}

	body.WriteString("We hope to see you there!\n\n")
	body.WriteString("Best regards,\nTicketly Team\n")

	return subject, body.String()
}

func (s *SubscriberService) buildSessionUpdateEmail(subscriber models.Subscriber, sessionUpdate *models.DebeziumSessionEvent) (string, string) {
	after := sessionUpdate.Payload.After
	before := sessionUpdate.Payload.Before
	operation := sessionUpdate.Payload.Operation

	timestamp := time.UnixMilli(sessionUpdate.Payload.Timestamp)

	var subject string
	var body strings.Builder

	if operation == "d" {
		if before == nil {
			return "", ""
		}

		subject = fmt.Sprintf("Session Cancelled: Session %s", before.ID)

		body.WriteString("Dear Subscriber,\n\n")
		body.WriteString("⚠️ IMPORTANT: A session you're subscribed to has been CANCELLED/DELETED:\n\n")

		body.WriteString("Cancelled Session Details:\n")
		body.WriteString(fmt.Sprintf("Session ID: %s\n", before.ID))
		body.WriteString(fmt.Sprintf("Event ID: %s\n", before.EventID))
		body.WriteString(fmt.Sprintf("Status: %s\n", before.Status))
		body.WriteString(fmt.Sprintf("Session Type: %s\n", before.SessionType))
		body.WriteString(fmt.Sprintf("Start Time: %s\n", time.Unix(before.StartTime/1000000, 0).Format("2006-01-02 15:04:05")))
		body.WriteString(fmt.Sprintf("End Time: %s\n", time.Unix(before.EndTime/1000000, 0).Format("2006-01-02 15:04:05")))
		body.WriteString(fmt.Sprintf("Cancelled: %s\n\n", timestamp.Format("2006-01-02 15:04:05")))

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
		if after == nil {
			return "", ""
		}

		subject = fmt.Sprintf("Session Update: Session %s", after.ID)

		body.WriteString("Dear Subscriber,\n\n")
		body.WriteString("A session you're subscribed to has been updated:\n\n")

		body.WriteString(fmt.Sprintf("Session ID: %s\n", after.ID))
		body.WriteString(fmt.Sprintf("Event ID: %s\n", after.EventID))
		body.WriteString(fmt.Sprintf("Status: %s\n", after.Status))
		body.WriteString(fmt.Sprintf("Session Type: %s\n", after.SessionType))
		body.WriteString(fmt.Sprintf("Start Time: %s\n", time.Unix(after.StartTime/1000000, 0).Format("2006-01-02 15:04:05")))
		body.WriteString(fmt.Sprintf("End Time: %s\n", time.Unix(after.EndTime/1000000, 0).Format("2006-01-02 15:04:05")))
		body.WriteString(fmt.Sprintf("Updated: %s\n\n", timestamp.Format("2006-01-02 15:04:05")))

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

			if before.VenueDetails != after.VenueDetails {
				body.WriteString("• Venue details updated\n")
			}
		} else if operation == "c" {
			body.WriteString("New Session Details:\n")
			if after.SalesStartTime > 0 {
				body.WriteString(fmt.Sprintf("• Sales Start: %s\n", time.Unix(after.SalesStartTime/1000000, 0).Format("2006-01-02 15:04:05")))
			}
		}
	}

	body.WriteString("\nBest regards,\nTicketly Team")

	return subject, body.String()
}
