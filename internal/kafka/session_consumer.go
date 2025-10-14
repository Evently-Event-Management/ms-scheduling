package kafka

import (
	"context"
	"encoding/json"
	"log"

	"ms-scheduling/internal/config"
	"ms-scheduling/internal/eventbridge"
	"ms-scheduling/internal/models"
	"ms-scheduling/internal/services"
	"time"
)

// SessionConsumer handles event session-related Kafka events
type SessionConsumer struct {
	BaseConsumer
	SchedulerService  *eventbridge.Service
	SubscriberService *services.SubscriberService
}

// NewSessionConsumer creates a new consumer for event session events
func NewSessionConsumer(cfg config.Config, schedulerService *eventbridge.Service, subscriberService *services.SubscriberService) *SessionConsumer {
	baseConsumer := NewBaseConsumer(cfg, cfg.KafkaURL, cfg.EventSessionsKafkaTopic)

	return &SessionConsumer{
		BaseConsumer:      *baseConsumer,
		SchedulerService:  schedulerService,
		SubscriberService: subscriberService,
	}
}

// StartConsuming starts consuming event session events
func (c *SessionConsumer) StartConsuming(ctx context.Context) error {
	log.Printf("Starting event session consumer for topic %s", c.Reader.Config().Topic)

	c.ConsumeMessages(ctx, c.processSessionEvent)

	return nil
}

// processSessionEvent handles event session events
func (c *SessionConsumer) processSessionEvent(value []byte) error {
	// Try to parse as DebeziumEvent
	var event models.DebeziumEvent
	if err := json.Unmarshal(value, &event); err != nil {
		log.Printf("Error unmarshalling Debezium event: %v", err)
		return err
	}

	// Handle both scheduling updates and notifications
	c.updateSessionSchedules(event)
	c.updateSessionNotification(event)

	return nil
}

// updateSessionNotification converts a real Debezium event to session update notification format
func (c *SessionConsumer) updateSessionNotification(event models.DebeziumEvent) {
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

// updateSessionSchedules handles scheduling updates for sessions
func (c *SessionConsumer) updateSessionSchedules(event models.DebeziumEvent) {
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

		// Schedule the session reminder email job (1 day before session starts)
		if after.StartTime > 0 {
			sessionStartTime := eventbridge.MicrosecondsToTime(after.StartTime)
			// Calculate 1 day before session start time
			reminderTime := sessionStartTime.AddDate(0, 0, -1) // Subtract 1 day

			salesStartTime := eventbridge.MicrosecondsToTime(after.SalesStartTime)
			reminderSalesStartTime := salesStartTime.Add(-30 * time.Minute)

			log.Printf("Scheduling session reminder email for session %s at %s (1 day before session starts)", after.ID, reminderTime.Format("2006-01-02 15:04:05"))
			log.Printf("Scheduling sales reminder email for session %s at %s (30 minutes before sales start)", after.ID, reminderSalesStartTime.Format("2006-01-02 15:04:05"))

			// Use the specialized reminder scheduler method with simplified parameters
			err := c.SchedulerService.CreateOrUpdateReminderSchedule(
				after.ID,
				reminderTime,
				"session-reminder-",
				"SESSION_START",
				"session reminder email job",
			)

			err_sale := c.SchedulerService.CreateOrUpdateReminderSchedule(
				after.ID,
				reminderSalesStartTime,
				"session-reminder-",
				"SALE_START",
				"sale reminder email job",
			)

			if err != nil {
				log.Printf("Error scheduling reminder email job for session %s: %v", after.ID, err)
			}

			if err_sale != nil {
				log.Printf("Error scheduling sales reminder email job for session %s: %v", after.ID, err_sale)
			}
		}

	case "u": // A session was updated
		log.Println("Handling update operation...")
		before, after := event.Payload.Before, event.Payload.After

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
			// Update the reminder email schedule (1 day before new start time)
			sessionStartTime := eventbridge.MicrosecondsToTime(after.StartTime)
			reminderTime := sessionStartTime.AddDate(0, 0, -1) // Subtract 1 day

			log.Printf("Session start time changed. Updating reminder email schedule for session %s to %s", after.ID, reminderTime.Format("2006-01-02 15:04:05"))

			// Use the specialized reminder scheduler method
			err := c.SchedulerService.CreateOrUpdateReminderSchedule(
				after.ID,
				reminderTime,
				"session-reminder-",
				"SESSION_START",
				"session reminder email job",
			)
			if err != nil {
				log.Printf("Error updating reminder email job for session %s: %v", after.ID, err)
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
