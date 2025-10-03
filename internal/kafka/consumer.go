// internal/kafka/consumer.go
package kafka

import (
	"context"
	"encoding/json"
	"log"
	"ms-scheduling/internal/models"
	"ms-scheduling/internal/scheduler"

	"github.com/segmentio/kafka-go"

	appconfig "ms-scheduling/internal/config"
)

// Consumer holds the dependencies for the Kafka consumer.
type Consumer struct {
	Reader           *kafka.Reader
	SchedulerService *scheduler.Service
	Config           appconfig.Config
}

// NewConsumer creates a new Kafka consumer with the given configuration.
func NewConsumer(cfg appconfig.Config, kafkaURL, topic string, schedulerService *scheduler.Service) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaURL},
		Topic:   topic,
		GroupID: "scheduler-service-group",
	})
	return &Consumer{
		Reader:           reader,
		SchedulerService: schedulerService,
		Config:           cfg,
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

		var event models.DebeziumEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Error unmarshalling Debezium event: %v", err)
			continue
		}

		c.processSessionChange(event)
	}
}

// processSessionChange is the main router for Debezium operations.
func (c *Consumer) processSessionChange(event models.DebeziumEvent) {
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
		// Schedule the on-sale job
		if after.SalesStartTime > 0 {
			onSaleTime := scheduler.MicrosecondsToTime(after.SalesStartTime)
			err := c.SchedulerService.CreateOrUpdateSchedule(
				after.ID,
				onSaleTime,
				"session-onsale-",
				c.Config.SQSONSaleQueueARN,
				"ON_SALE",
				"on-sale job",
			)
			if err != nil {
				log.Printf("Error scheduling on-sale job for session %s: %v", after.ID, err)
			}
		}
		// Schedule the session-closed job
		if after.EndTime > 0 {
			closedTime := scheduler.MicrosecondsToTime(after.EndTime)
			err := c.SchedulerService.CreateOrUpdateSchedule(
				after.ID,
				closedTime,
				"session-closed-",
				c.Config.SQSClosedQueueARN,
				"CLOSED",
				"closed job",
			)
			if err != nil {
				log.Printf("Error scheduling closed job for session %s: %v", after.ID, err)
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
			return
		}

		// Check if on-sale time changed
		if after.SalesStartTime != before.SalesStartTime {
			onSaleTime := scheduler.MicrosecondsToTime(after.SalesStartTime)
			log.Printf("Sales start time for session %s changed. Updating schedule.", after.ID)
			err := c.SchedulerService.CreateOrUpdateSchedule(
				after.ID,
				onSaleTime,
				"session-onsale-",
				c.Config.SQSONSaleQueueARN,
				"ON_SALE",
				"on-sale job",
			)
			if err != nil {
				log.Printf("Error updating on-sale job for session %s: %v", after.ID, err)
			}
		}
		// Check if end time changed
		if after.EndTime != before.EndTime {
			closedTime := scheduler.MicrosecondsToTime(after.EndTime)
			log.Printf("End time for session %s changed. Updating schedule.", after.ID)
			err := c.SchedulerService.CreateOrUpdateSchedule(
				after.ID,
				closedTime,
				"session-closed-",
				c.Config.SQSClosedQueueARN,
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
	}
}
