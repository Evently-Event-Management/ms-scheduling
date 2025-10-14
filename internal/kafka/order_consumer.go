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
	CreatedConsumer   BaseConsumer
	UpdatedConsumer   BaseConsumer
	CancelledConsumer BaseConsumer
	SubscriberService *services.SubscriberService
}

// NewOrderConsumer creates a new consumer for order events
func NewOrderConsumer(cfg config.Config, subscriberService *services.SubscriberService) *OrderConsumer {
	result := &OrderConsumer{
		SubscriberService: subscriberService,
	}

	// Only create consumers for non-empty topics
	if cfg.OrdersKafkaTopic != "" {
		createdConsumer := NewBaseConsumer(cfg, cfg.KafkaURL, cfg.OrdersKafkaTopic)
		result.CreatedConsumer = *createdConsumer
	}

	if cfg.OrdersUpdatedKafkaTopic != "" {
		updatedConsumer := NewBaseConsumer(cfg, cfg.KafkaURL, cfg.OrdersUpdatedKafkaTopic)
		result.UpdatedConsumer = *updatedConsumer
	}

	if cfg.OrdersCancelledKafkaTopic != "" {
		cancelledConsumer := NewBaseConsumer(cfg, cfg.KafkaURL, cfg.OrdersCancelledKafkaTopic)
		result.CancelledConsumer = *cancelledConsumer
	}

	return result
}

// StartConsuming starts consuming order events
func (c *OrderConsumer) StartConsuming(ctx context.Context) error {
	// Start a goroutine for each configured topic

	// Check if we have any consumers configured
	if c.CreatedConsumer.Reader == nil && c.UpdatedConsumer.Reader == nil && c.CancelledConsumer.Reader == nil {
		log.Println("No order Kafka topics configured, skipping order consumer setup")
		return nil
	}

	// Created orders
	if c.CreatedConsumer.Reader != nil {
		go func() {
			log.Printf("Starting order created consumer for topic %s", c.CreatedConsumer.Reader.Config().Topic)
			c.CreatedConsumer.ConsumeMessages(ctx, c.processOrderCreated)
		}()
	}

	// Updated orders
	if c.UpdatedConsumer.Reader != nil {
		go func() {
			log.Printf("Starting order updated consumer for topic %s", c.UpdatedConsumer.Reader.Config().Topic)
			c.UpdatedConsumer.ConsumeMessages(ctx, c.processOrderUpdated)
		}()
	}

	// Cancelled orders
	if c.CancelledConsumer.Reader != nil {
		go func() {
			log.Printf("Starting order cancelled consumer for topic %s", c.CancelledConsumer.Reader.Config().Topic)
			c.CancelledConsumer.ConsumeMessages(ctx, c.processOrderCancelled)
		}()
	}

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

// processOrderUpdated handles ticketly.order.updated events
func (c *OrderConsumer) processOrderUpdated(value []byte) error {
	var order services.OrderCreatedEvent
	if err := json.Unmarshal(value, &order); err != nil {
		log.Printf("Error unmarshalling order.updated event: %v", err)
		return err
	}
	log.Printf("Processing order.updated for OrderID=%s UserID=%s", order.OrderID, order.UserID)

	// Get or create subscriber
	subscriber, err := c.SubscriberService.GetOrCreateSubscriber(order.UserID)
	if err != nil {
		log.Printf("Error getting/creating subscriber for user %s: %v", order.UserID, err)
		return err
	}

	// For orders changing to 'completed' status, add subscriptions
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
	}

	// Send appropriate order email based on status
	if err := c.SubscriberService.SendOrderConfirmationEmail(subscriber, &order); err != nil {
		log.Printf("Error sending order email: %v", err)
		return err
	}

	log.Printf("Successfully processed updated order %s for user %s (email: %s)",
		order.OrderID, order.UserID, subscriber.SubscriberMail)

	return nil
}

// processOrderCancelled handles ticketly.order.cancelled events
func (c *OrderConsumer) processOrderCancelled(value []byte) error {
	var order services.OrderCreatedEvent
	if err := json.Unmarshal(value, &order); err != nil {
		log.Printf("Error unmarshalling order.cancelled event: %v", err)
		return err
	}
	log.Printf("Processing order.cancelled for OrderID=%s UserID=%s", order.OrderID, order.UserID)

	// Get subscriber - don't create if doesn't exist
	subscriber, err := c.SubscriberService.GetSubscriberByUserID(order.UserID)
	if err != nil {
		log.Printf("Error getting subscriber for user %s: %v", order.UserID, err)
		return err
	}

	if subscriber == nil {
		log.Printf("No subscriber found for user %s - skipping cancelled order notification", order.UserID)
		return nil
	}

	// Force the status to cancelled for the email
	order.Status = "cancelled"

	// Send cancellation email
	if err := c.SubscriberService.SendOrderConfirmationEmail(subscriber, &order); err != nil {
		log.Printf("Error sending order cancellation email: %v", err)
		return err
	}

	log.Printf("Successfully processed cancelled order %s for user %s (email: %s)",
		order.OrderID, order.UserID, subscriber.SubscriberMail)

	return nil
}
