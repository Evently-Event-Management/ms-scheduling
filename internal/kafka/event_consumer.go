package kafka

import (
	"context"
	"encoding/json"
	"log"

	"ms-scheduling/internal/config"
	"ms-scheduling/internal/models"
	"ms-scheduling/internal/services"
)

// EventConsumer handles event-related Kafka events
type EventConsumer struct {
	BaseConsumer
	SubscriberService *services.SubscriberService
}

// NewEventConsumer creates a new consumer for event events
func NewEventConsumer(cfg config.Config, subscriberService *services.SubscriberService) *EventConsumer {
	baseConsumer := NewBaseConsumer(cfg, cfg.KafkaURL, cfg.EventsKafkaTopic)

	return &EventConsumer{
		BaseConsumer:      *baseConsumer,
		SubscriberService: subscriberService,
	}
}

// StartConsuming starts consuming event events
func (c *EventConsumer) StartConsuming(ctx context.Context) error {
	log.Printf("Starting event consumer for topic %s", c.Reader.Config().Topic)

	c.ConsumeMessages(ctx, c.processEventEvent)

	return nil
}

// processEventEvent handles event events
func (c *EventConsumer) processEventEvent(value []byte) error {
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
		return err
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
	case "c": // Skip event creation notification
		log.Printf("Skipping notification for event creation: %s", eventID)

	case "u": // Event update - notify only if before=PENDING and after=APPROVED
		// Check status transition for updates
		if rawEvent.Payload.Before != nil && rawEvent.Payload.After != nil {
			beforeStatus := rawEvent.Payload.Before.Status
			afterStatus := rawEvent.Payload.After.Status

			if beforeStatus == "PENDING" && afterStatus == "APPROVED" {
				// This is a status change from PENDING to APPROVED - treat as creation
				if err := c.SubscriberService.ProcessEventCreation(&eventEvent); err != nil {
					log.Printf("Error processing event approval notification from Debezium: %v", err)
					return err
				}
				log.Printf("Successfully processed event approval (PENDING->APPROVED) notification for event %s", eventID)
			} else if afterStatus == "APPROVED" {
				// Other changes but final status is still APPROVED - process as update
				if err := c.SubscriberService.ProcessEventUpdate(&eventEvent); err != nil {
					log.Printf("Error processing event update notification from Debezium: %v", err)
					return err
				}
				log.Printf("Successfully processed event update notification for approved event %s", eventID)
			} else {
				// Status is not APPROVED - skip notification
				log.Printf("Skipping notification for event %s - status is %s (not APPROVED)", eventID, afterStatus)
			}
		}

	case "d": // Event deletion - process normally for subscribers
		if err := c.SubscriberService.ProcessEventUpdate(&eventEvent); err != nil {
			log.Printf("Error processing event deletion notification from Debezium: %v", err)
			return err
		}
		log.Printf("Successfully processed event deletion notification for event %s", eventID)

	default:
		log.Printf("Unhandled operation '%s' for event %s", rawEvent.Payload.Op, eventID)
	}

	return nil
}
