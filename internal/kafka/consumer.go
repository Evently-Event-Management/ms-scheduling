// internal/kafka/consumer.go
package kafka

import (
	"context"
	"encoding/json"
	"log"
	"ms-scheduling/internal/eventbridge"
	"ms-scheduling/internal/models"
	"ms-scheduling/internal/services"

	"github.com/segmentio/kafka-go"

	appconfig "ms-scheduling/internal/config"
)

// Consumer holds the dependencies for the Kafka consumer.
type Consumer struct {
	Reader            *kafka.Reader
	SchedulerService  *eventbridge.Service
	SubscriberService *services.SubscriberService
	Config            appconfig.Config
}

// NewConsumer creates a new Kafka consumer with the given configuration.
func NewConsumer(cfg appconfig.Config, kafkaURL, topic string, schedulerService *eventbridge.Service, subscriberService *services.SubscriberService) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaURL},
		Topic:   topic,
		GroupID: "scheduler-service-group",
	})
	return &Consumer{
		Reader:            reader,
		SchedulerService:  schedulerService,
		SubscriberService: subscriberService,
		Config:            cfg,
	}
}

// ConsumeDebeziumEvents starts consuming Debezium events from Kafka.
func (c *Consumer) ConsumeDebeziumEvents() {
	for {
		msg, err := c.Reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Error reading from Kafka: %v", err)
			continue
		}

		log.Printf("Received Kafka message from topic %s", msg.Topic)

		// Handle order created events
		if msg.Topic == "ticketly.order.created" {
			c.processOrderCreated(msg.Value)
			continue
		}

		// Handle session update notifications and Debezium events
		if msg.Topic == "dbz.ticketly.public.event_sessions" {
			// Try to parse as DebeziumEvent first (this is the real Debezium format)
			var event models.DebeziumEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Error unmarshalling Debezium event: %v", err)
				continue
			}

			c.updateSessionSchedules(event)    // Schedule jobs based on session changes
			c.updateSessionNotification(event) // Instant notification for session updates
			continue
		}

		// Handle event update notifications and Debezium events
		if msg.Topic == "dbz.ticketly.public.events" {
			// Process event update notification (email logic)
			c.processEventNotification(msg.Value)
			continue
		}
		var event models.DebeziumEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Error unmarshalling Debezium event: %v", err)
			continue
		}
	}
}

// OrderCreatedEvent represents the structure of the order.created Kafka event
type OrderCreatedEvent struct {
	OrderID        string  `json:"OrderID"`
	UserID         string  `json:"UserID"`
	EventID        string  `json:"EventID"`
	SessionID      string  `json:"SessionID"`
	Status         string  `json:"Status"`
	SubTotal       float64 `json:"SubTotal"`
	DiscountID     string  `json:"DiscountID"`
	DiscountCode   string  `json:"DiscountCode"`
	DiscountAmount float64 `json:"DiscountAmount"`
	Price          float64 `json:"Price"`
	CreatedAt      string  `json:"CreatedAt"`
	PaymentAT      string  `json:"PaymentAT"`
	Tickets        []struct {
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
	} `json:"tickets"`
}

// processOrderCreated handles ticketly.order.created events
func (c *Consumer) processOrderCreated(value []byte) {
	var order services.OrderCreatedEvent
	if err := json.Unmarshal(value, &order); err != nil {
		log.Printf("Error unmarshalling order.created event: %v", err)
		return
	}
	log.Printf("Processing order.created for OrderID=%s UserID=%s", order.OrderID, order.UserID)

	// Get or create subscriber
	subscriber, err := c.SubscriberService.GetOrCreateSubscriber(order.UserID)
	if err != nil {
		log.Printf("Error getting/creating subscriber for user %s: %v", order.UserID, err)
		return
	}

	// Add subscription to the event and session
	c.SubscriberService.AddSubscription(subscriber.SubscriberID, models.SubscriptionCategoryEvent, order.EventID)
	c.SubscriberService.AddSubscription(subscriber.SubscriberID, models.SubscriptionCategorySession, order.SessionID)

	// Send order confirmation email
	if err := c.SubscriberService.SendOrderConfirmationEmail(subscriber, &order); err != nil {
		log.Printf("Error sending order confirmation email: %v", err)
		return
	}

	log.Printf("Successfully processed order %s for user %s (email: %s)",
		order.OrderID, order.UserID, subscriber.SubscriberMail)
}

// processSessionUpdateNotification handles session update notifications from Debezium
func (c *Consumer) processSessionUpdateNotification(value []byte) {
	var sessionEvent models.DebeziumSessionEvent
	if err := json.Unmarshal(value, &sessionEvent); err != nil {
		log.Printf("Error unmarshalling session update event: %v", err)
		return
	}

	log.Printf("Processing session update notification for operation: %s", sessionEvent.Payload.Operation)

	// Process the session update notification
	if err := c.SubscriberService.ProcessSessionUpdate(&sessionEvent); err != nil {
		log.Printf("Error processing session update notification: %v", err)
		return
	}

	log.Printf("Successfully processed session update notification")
}

// updateSessionNotification converts a real Debezium event to session update notification format
func (c *Consumer) updateSessionNotification(event models.DebeziumEvent) {
	log.Printf("Processing session update notification from real Debezium event, operation: %s", event.Payload.Op)

	// Determine session ID for logging
	sessionID := ""
	if event.Payload.After != nil {
		sessionID = event.Payload.After.ID
	} else if event.Payload.Before != nil {
		sessionID = event.Payload.Before.ID
	}

	log.Printf("Processing session %s notification for operation: %s", sessionID, event.Payload.Op)

	// Convert DebeziumEvent to DebeziumSessionEvent for notification processing
	sessionEvent := models.DebeziumSessionEvent{
		Payload: models.SessionUpdate{
			Before:    event.Payload.Before,
			After:     event.Payload.After,
			Source:    event.Payload.Source,
			Operation: event.Payload.Op,
			Timestamp: event.Payload.TsMs,
			SessionID: sessionID,
		},
	}

	// Process the session update notification
	if err := c.SubscriberService.ProcessSessionUpdate(&sessionEvent); err != nil {
		log.Printf("Error processing session update notification from Debezium: %v", err)
		return
	}

	log.Printf("Successfully processed session update notification from Debezium event for session %s", sessionID)
}

// processEventNotification processes Debezium events for the events table
func (c *Consumer) processEventNotification(value []byte) {
	log.Printf("Processing event update notification from Debezium")

	// Parse the raw JSON into a generic structure to extract event data
	var rawEvent struct {
		Payload struct {
			Before *models.Event         `json:"before"`
			After  *models.Event         `json:"after"`
			Source models.DebeziumSource `json:"source"`
			Op     string                `json:"op"`
			TsMs   int64                 `json:"ts_ms"`
		} `json:"payload"`
	}

	if err := json.Unmarshal(value, &rawEvent); err != nil {
		log.Printf("Error unmarshalling event Debezium data: %v", err)
		return
	}

	// Determine event ID for logging
	eventID := ""
	if rawEvent.Payload.After != nil {
		eventID = rawEvent.Payload.After.ID
	} else if rawEvent.Payload.Before != nil {
		eventID = rawEvent.Payload.Before.ID
	}

	log.Printf("Processing event %s notification for operation: %s", eventID, rawEvent.Payload.Op)

	// Convert to DebeziumEventEvent for notification processing
	eventEvent := models.DebeziumEventEvent{
		Payload: models.EventUpdate{
			Before:    rawEvent.Payload.Before,
			After:     rawEvent.Payload.After,
			Source:    rawEvent.Payload.Source,
			Operation: rawEvent.Payload.Op,
			Timestamp: rawEvent.Payload.TsMs,
			EventID:   eventID,
		},
	}

	// Handle different operations
	switch rawEvent.Payload.Op {
	case "c": // Event creation - notify organization subscribers
		if err := c.SubscriberService.ProcessEventCreation(&eventEvent); err != nil {
			log.Printf("Error processing event creation notification from Debezium: %v", err)
			return
		}
		log.Printf("Successfully processed event creation notification for event %s", eventID)

	case "u", "d": // Event update/delete - notify event subscribers
		if err := c.SubscriberService.ProcessEventUpdate(&eventEvent); err != nil {
			log.Printf("Error processing event update notification from Debezium: %v", err)
			return
		}
		log.Printf("Successfully processed event update notification for event %s", eventID)

	default:
		log.Printf("Unhandled operation '%s' for event %s", rawEvent.Payload.Op, eventID)
	}
}

// updateSessionSchedules is the main router for Debezium operations.
func (c *Consumer) updateSessionSchedules(event models.DebeziumEvent) {
	sessionID := ""
	if event.Payload.After != nil {
		sessionID = event.Payload.After.ID
	} else if event.Payload.Before != nil {
		sessionID = event.Payload.Before.ID // For delete operations
	}

	if sessionID == "" {
		log.Println("Could not determine session ID from Debezium event. Skipping.")
		return
	}

	log.Printf("Processing operation '%s' for session ID: %s", event.Payload.Op, sessionID)

	switch event.Payload.Op {
	case "c": // A new session was created
		log.Println("Handling create operation...")
		after := event.Payload.After
		// Schedule the on-sale job using standard scheduler
		if after.SalesStartTime > 0 {
			onSaleTime := eventbridge.MicrosecondsToTime(after.SalesStartTime)
			err := c.SchedulerService.CreateOrUpdateSchedule(
				after.ID,
				onSaleTime,
				"session-onsale-",
				"ON_SALE",
				"on-sale job",
			)
			if err != nil {
				log.Printf("Error scheduling on-sale job for session %s: %v", after.ID, err)
			}
		}
		// Schedule the session-closed job using standard scheduler
		if after.EndTime > 0 {
			closedTime := eventbridge.MicrosecondsToTime(after.EndTime)
			err := c.SchedulerService.CreateOrUpdateSchedule(
				after.ID,
				closedTime,
				"session-closed-",
				"CLOSED",
				"closed job",
			)
			if err != nil {
				log.Printf("Error scheduling closed job for session %s: %v", after.ID, err)
			}
		}
		// Schedule the session reminder email job (1 day before session starts) using reminder-specific scheduler
		if after.StartTime > 0 {
			sessionStartTime := eventbridge.MicrosecondsToTime(after.StartTime)
			// Calculate 1 day before session start time
			reminderTime := sessionStartTime.AddDate(0, 0, -1) // Subtract 1 day

			log.Printf("Scheduling session reminder email for session %s at %s (1 day before session starts)", after.ID, reminderTime.Format("2006-01-02 15:04:05"))

			// Use the specialized reminder scheduler method
			err := c.SchedulerService.CreateOrUpdateReminderSchedule(
				after.ID,
				reminderTime,
				"session-reminder-",
				"REMINDER_EMAIL",
				"SESSION_START",
				"session reminder email job",
			)
			if err != nil {
				log.Printf("Error scheduling reminder email job for session %s: %v", after.ID, err)
			} else {
				log.Printf("Successfully scheduled reminder email for session %s to be sent on %s", after.ID, reminderTime.Format("2006-01-02 15:04:05"))
			}
		}

	case "u": // A session was updated
		log.Println("Handling update operation...")
		before, after := event.Payload.Before, event.Payload.After
		log.Printf("Before: %+v", before)
		log.Printf("After: %+v", after)
		// Sanity check
		if before == nil || after == nil {
			return
		}

		// If status changed to CANCELLED, delete schedules
		if after.Status == "CANCELLED" && before.Status != "CANCELLED" {
			log.Printf("Session %s was cancelled. Deleting schedules.", after.ID)
			c.SchedulerService.DeleteSchedule(after.ID, "session-onsale-")
			c.SchedulerService.DeleteSchedule(after.ID, "session-closed-")
			c.SchedulerService.DeleteSchedule(after.ID, "session-reminder-")
			log.Printf("Deleted all schedules (including reminder email) for cancelled session %s", after.ID)
			return
		}

		if after.Status == "CANCELLED" {
			log.Printf("Session %s was cancelled. No further scheduling actions will be taken.", after.ID)
			return
		}

		// Check if on-sale time changed
		if after.SalesStartTime != before.SalesStartTime {
			onSaleTime := eventbridge.MicrosecondsToTime(after.SalesStartTime)
			log.Printf("Sales start time for session %s changed. Updating schedule.", after.ID)
			err := c.SchedulerService.CreateOrUpdateSchedule(
				after.ID,
				onSaleTime,
				"session-onsale-",
				"ON_SALE",
				"on-sale job",
			)
			if err != nil {
				log.Printf("Error updating on-sale job for session %s: %v", after.ID, err)
			}
		}
		// Check if start time changed
		if after.StartTime != before.StartTime {
			// Update the reminder email schedule (1 day before new start time) using reminder-specific scheduler
			sessionStartTime := eventbridge.MicrosecondsToTime(after.StartTime)
			reminderTime := sessionStartTime.AddDate(0, 0, -1) // Subtract 1 day

			log.Printf("Session start time changed. Updating reminder email schedule for session %s to %s", after.ID, reminderTime.Format("2006-01-02 15:04:05"))

			// Use the specialized reminder scheduler method
			err := c.SchedulerService.CreateOrUpdateReminderSchedule(
				after.ID,
				reminderTime,
				"session-reminder-",
				"REMINDER_EMAIL",
				"SESSION_START",
				"session reminder email job",
			)
			if err != nil {
				log.Printf("Error updating reminder email job for session %s: %v", after.ID, err)
			} else {
				log.Printf("Successfully updated reminder email schedule for session %s", after.ID)
			}
		}

		if after.EndTime != before.EndTime {
			closedTime := eventbridge.MicrosecondsToTime(after.EndTime)
			log.Printf("End time for session %s changed. Updating schedule.", after.ID)
			err := c.SchedulerService.CreateOrUpdateSchedule(
				after.ID,
				closedTime,
				"session-closed-",
				"CLOSED",
				"closed job",
			)
			if err != nil {
				log.Printf("Error updating closed job for session %s: %v", after.ID, err)
			}
		}

	case "d": // A session was deleted
		log.Println("Handling delete operation...")
		before := event.Payload.Before
		if before == nil {
			return
		}
		c.SchedulerService.DeleteSchedule(before.ID, "session-onsale-")
		c.SchedulerService.DeleteSchedule(before.ID, "session-closed-")
		c.SchedulerService.DeleteSchedule(before.ID, "session-reminder-")
		log.Printf("Deleted all schedules (including reminder email) for deleted session %s", before.ID)
	}
}
