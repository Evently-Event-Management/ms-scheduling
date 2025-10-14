package kafka

import (
	"context"
	"encoding/json"
	"log"

	"ms-scheduling/internal/config"
	"ms-scheduling/internal/models"
	"ms-scheduling/internal/services"
)

// OrderConsumer handles order-related Kafka events
type OrderConsumer struct {
	BaseConsumer
	SubscriberService *services.SubscriberService
}

// NewOrderConsumer creates a new consumer for order events
func NewOrderConsumer(cfg config.Config, subscriberService *services.SubscriberService) *OrderConsumer {
	baseConsumer := NewBaseConsumer(cfg, cfg.KafkaURL, cfg.OrdersKafkaTopic)

	return &OrderConsumer{
		BaseConsumer:      *baseConsumer,
		SubscriberService: subscriberService,
	}
}

// StartConsuming starts consuming order events
func (c *OrderConsumer) StartConsuming(ctx context.Context) error {
	log.Printf("Starting order consumer for topic %s", c.Reader.Config().Topic)

	c.ConsumeMessages(ctx, c.processOrderCreated)

	return nil
}

// processOrderCreated handles ticketly.order.created events
func (c *OrderConsumer) processOrderCreated(value []byte) error {
	var order services.OrderCreatedEvent
	if err := json.Unmarshal(value, &order); err != nil {
		log.Printf("Error unmarshalling order.created event: %v", err)
		return err
	}
	log.Printf("Processing order.created for OrderID=%s UserID=%s", order.OrderID, order.UserID)

	// Get or create subscriber
	subscriber, err := c.SubscriberService.GetOrCreateSubscriber(order.UserID)
	if err != nil {
		log.Printf("Error getting/creating subscriber for user %s: %v", order.UserID, err)
		return err
	}

	// Only add subscriptions for orders in 'completed' status
	// For pending orders, we'll add subscriptions when they're completed
	if order.Status == "completed" {
		// Add subscription to the event and session
		if err := c.SubscriberService.AddSubscription(subscriber.SubscriberID, models.SubscriptionCategoryEvent, order.EventID); err != nil {
			log.Printf("Error adding event subscription: %v", err)
		}

		if err := c.SubscriberService.AddSubscription(subscriber.SubscriberID, models.SubscriptionCategorySession, order.SessionID); err != nil {
			log.Printf("Error adding session subscription: %v", err)
		}

		if order.OrganizationID != "" {
			if err := c.SubscriberService.AddSubscription(subscriber.SubscriberID, models.SubscriptionCategoryOrganization, order.OrganizationID); err != nil {
				log.Printf("Error adding organization subscription: %v", err)
			}
		}

		log.Printf("Added subscriptions for completed order %s", order.OrderID)
	} else {
		log.Printf("Order %s has status '%s' - subscriptions will be added when completed", order.OrderID, order.Status)
	}

	// Send appropriate order email based on status
	if err := c.SubscriberService.SendOrderConfirmationEmail(subscriber, &order); err != nil {
		log.Printf("Error sending order email: %v", err)
		return err
	}

	log.Printf("Successfully processed order %s for user %s (email: %s)",
		order.OrderID, order.UserID, subscriber.SubscriberMail)

	return nil
}
