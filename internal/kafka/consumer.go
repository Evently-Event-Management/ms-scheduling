// internal/kafka/consumer.go
package kafka

import (
	"context"
	"encoding/json"
	"log"
	"ms-scheduling/internal/models"
	"ms-scheduling/internal/scheduler"
	"ms-scheduling/internal/services"
	"strconv"

	"github.com/segmentio/kafka-go"

	appconfig "ms-scheduling/internal/config"
)

// Consumer holds the dependencies for the Kafka consumer.
type Consumer struct {
	Reader            *kafka.Reader
	SchedulerService  *scheduler.Service
	SubscriberService *services.SubscriberService
	Config            appconfig.Config
}

// NewConsumer creates a new Kafka consumer with the given configuration.
func NewConsumer(cfg appconfig.Config, kafkaURL, topic string, schedulerService *scheduler.Service, subscriberService *services.SubscriberService) *Consumer {
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

		var event models.DebeziumEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Error unmarshalling Debezium event: %v", err)
			continue
		}

		c.processSessionChange(event)
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
	if eventID, err := strconv.Atoi(order.EventID); err == nil {
		c.SubscriberService.AddSubscription(subscriber.SubscriberID, models.SubscriptionCategoryEvent, eventID)
	}
	if sessionID, err := strconv.Atoi(order.SessionID); err == nil {
		c.SubscriberService.AddSubscription(subscriber.SubscriberID, models.SubscriptionCategorySession, sessionID)
	}

	// Send order confirmation email
	if err := c.SubscriberService.SendOrderConfirmationEmail(subscriber, &order); err != nil {
		log.Printf("Error sending order confirmation email: %v", err)
		return
	}

	log.Printf("Successfully processed order %s for user %s (email: %s)",
		order.OrderID, order.UserID, subscriber.SubscriberMail)
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
				c.Config.SQSSessionSchedulingQueueARN,
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
				c.Config.SQSSessionSchedulingQueueARN,
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

		if after.Status == "CANCELLED" {
			log.Printf("Session %s was cancelled. No further scheduling actions will be taken.", after.ID)
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
				c.Config.SQSSessionSchedulingQueueARN,
				"ON_SALE",
				"on-sale job",
			)
			if err != nil {
				log.Printf("Error updating on-sale job for session %s: %v", after.ID, err)
			}
		}
		// Check if start time changed
		if after.StartTime != before.StartTime {
			closedTime := scheduler.MicrosecondsToTime(after.StartTime)
			log.Printf("Start time for session %s changed. Updating schedule.", after.ID)
			err := c.SchedulerService.CreateOrUpdateSchedule(
				after.ID,
				closedTime,
				"session-closed-",
				c.Config.SQSSessionSchedulingQueueARN,
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
