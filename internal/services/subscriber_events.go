package services

import (
	"fmt"
	"log"
	"strings"
	"time"

	"ms-scheduling/internal/models"
)

func (s *SubscriberService) ProcessEventUpdate(eventUpdate *models.DebeziumEventEvent) error {
	log.Printf("Processing event update event: %s", eventUpdate.Payload.Operation)

	if eventUpdate.Payload.Operation == "r" {
		log.Printf("Skipping event update for initial snapshot operation: %s", eventUpdate.Payload.Operation)
		return nil
	}

	var eventID string
	if eventUpdate.Payload.Operation == "d" {
		if eventUpdate.Payload.Before != nil {
			eventID = eventUpdate.Payload.Before.ID
		} else {
			return fmt.Errorf("no before data available for event deletion")
		}
	} else {
		if eventUpdate.Payload.After != nil {
			eventID = eventUpdate.Payload.After.ID
		} else {
			return fmt.Errorf("no after data available for event update")
		}
	}

	subscribers, err := s.GetEventSubscribers(eventID)
	if err != nil {
		return fmt.Errorf("error getting event subscribers: %w", err)
	}

	if len(subscribers) == 0 {
		log.Printf("No subscribers found for event ID: %s", eventID)
		return nil
	}

	return s.SendEventUpdateEmails(subscribers, eventUpdate)
}

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

func (s *SubscriberService) SendEventUpdateEmails(subscribers []models.Subscriber, eventUpdate *models.DebeziumEventEvent) error {
	log.Printf("Sending event update emails to %d subscribers", len(subscribers))

	operation := eventUpdate.Payload.Operation
	before := eventUpdate.Payload.Before
	after := eventUpdate.Payload.After

	// Get organization name for context
	organizationName := "Organization"
	if after != nil && after.OrganizationID != "" {
		organizationName = s.getOrganizationName(after.OrganizationID)
	} else if before != nil && before.OrganizationID != "" {
		organizationName = s.getOrganizationName(before.OrganizationID)
	}

	for _, subscriber := range subscribers {
		var err error

		switch operation {
		case "d": // Deletion/Cancellation
			if before != nil && s.EmailManager != nil {
				err = s.EmailManager.SendEventCancelledEmail(subscriber.SubscriberMail, before, organizationName)
			} else {
				// Fallback to old method
				subject, body := s.buildEventUpdateEmail(subscriber, eventUpdate)
				err = s.EmailService.SendEmail(subscriber.SubscriberMail, subject, body)
			}
		case "u": // Update
			if before != nil && after != nil && s.EmailManager != nil {
				err = s.EmailManager.SendEventUpdatedEmail(subscriber.SubscriberMail, before, after, organizationName)
			} else {
				// Fallback to old method
				subject, body := s.buildEventUpdateEmail(subscriber, eventUpdate)
				err = s.EmailService.SendEmail(subscriber.SubscriberMail, subject, body)
			}
		default:
			// For other operations, use old method
			subject, body := s.buildEventUpdateEmail(subscriber, eventUpdate)
			err = s.EmailService.SendEmail(subscriber.SubscriberMail, subject, body)
		}

		if err != nil {
			log.Printf("Error sending event update email to %s: %v", subscriber.SubscriberMail, err)
			continue
		}

		log.Printf("Event update email sent successfully to: %s", subscriber.SubscriberMail)
	}

	return nil
}

func (s *SubscriberService) buildEventUpdateEmail(subscriber models.Subscriber, eventUpdate *models.DebeziumEventEvent) (string, string) {
	after := eventUpdate.Payload.After
	before := eventUpdate.Payload.Before
	operation := eventUpdate.Payload.Operation

	timestamp := time.UnixMilli(eventUpdate.Payload.Timestamp)

	var subject string
	var body strings.Builder

	if operation == "d" {
		if before == nil {
			return "", ""
		}

		subject = fmt.Sprintf("Event Cancelled: %s", before.Title)

		body.WriteString("Dear Subscriber,\n\n")
		body.WriteString("⚠️ IMPORTANT: An event you're subscribed to has been CANCELLED/DELETED:\n\n")

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
		if after == nil {
			return "", ""
		}

		subject = fmt.Sprintf("Event Update: %s", after.Title)

		body.WriteString("Dear Subscriber,\n\n")
		body.WriteString("An event you're subscribed to has been updated:\n\n")

		body.WriteString(fmt.Sprintf("Event ID: %s\n", after.ID))
		body.WriteString(fmt.Sprintf("Title: %s\n", after.Title))
		body.WriteString(fmt.Sprintf("Description: %s\n", after.Description))
		body.WriteString(fmt.Sprintf("Status: %s\n", after.Status))
		body.WriteString(fmt.Sprintf("Created: %s\n", time.Unix(after.CreatedAt/1000000, 0).Format("2006-01-02 15:04:05")))
		body.WriteString(fmt.Sprintf("Updated: %s\n\n", timestamp.Format("2006-01-02 15:04:05")))

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
			body.WriteString("New Event Details:\n")
			body.WriteString(fmt.Sprintf("• Status: %s\n", after.Status))
			if after.Overview != "" {
				body.WriteString(fmt.Sprintf("• Overview: %s\n", after.Overview))
			}
		}

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

func (s *SubscriberService) GetOrganizationSubscribers(organizationID string) ([]models.Subscriber, error) {
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

func (s *SubscriberService) ProcessEventCreation(eventUpdate *models.DebeziumEventEvent) error {
	log.Printf("Processing event creation notification: %s", eventUpdate.Payload.Operation)

	if eventUpdate.Payload.After == nil {
		return fmt.Errorf("no after data available for event creation")
	}

	organizationID := eventUpdate.Payload.After.OrganizationID
	eventID := eventUpdate.Payload.After.ID

	log.Printf("Processing event creation for event %s in organization %s", eventID, organizationID)

	subscribers, err := s.GetOrganizationSubscribers(organizationID)
	if err != nil {
		return fmt.Errorf("error getting organization subscribers: %w", err)
	}

	if len(subscribers) == 0 {
		log.Printf("No organization subscribers found for organization ID: %s", organizationID)
		return nil
	}

	log.Printf("Found %d subscribers for organization %s", len(subscribers), organizationID)

	return s.SendEventCreationEmails(subscribers, eventUpdate)
}

func (s *SubscriberService) SendEventCreationEmails(subscribers []models.Subscriber, eventUpdate *models.DebeziumEventEvent) error {
	log.Printf("Sending event creation emails to %d subscribers", len(subscribers))

	after := eventUpdate.Payload.After
	if after == nil {
		return nil
	}

	// Get organization name for context
	organizationName := s.getOrganizationName(after.OrganizationID)

	for _, subscriber := range subscribers {
		var err error

		// Check if this is an approval (PENDING -> APPROVED) or initial creation with APPROVED status
		if after.Status == "APPROVED" && s.EmailManager != nil {
			err = s.EmailManager.SendEventCreatedEmail(subscriber.SubscriberMail, after, organizationName)
		} else {
			// Fallback to old method or skip if not approved
			subject, body := s.buildEventCreationEmail(subscriber, eventUpdate)
			if subject != "" {
				err = s.EmailService.SendEmail(subscriber.SubscriberMail, subject, body)
			}
		}

		if err != nil {
			log.Printf("Error sending event creation email to %s: %v", subscriber.SubscriberMail, err)
			continue
		}

		log.Printf("Event creation email sent successfully to: %s", subscriber.SubscriberMail)
	}

	return nil
}

func (s *SubscriberService) buildEventCreationEmail(subscriber models.Subscriber, eventUpdate *models.DebeziumEventEvent) (string, string) {
	after := eventUpdate.Payload.After

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
