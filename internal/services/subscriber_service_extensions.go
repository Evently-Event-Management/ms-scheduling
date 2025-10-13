package services

import (
	"fmt"
	"log"
	"ms-scheduling/internal/models"
)

// RemoveSubscription removes a subscription for a subscriber
func (s *SubscriberService) RemoveSubscription(subscriberID int, category models.SubscriptionCategory, targetUUID string) error {
	query := `
		DELETE FROM subscriptions 
		WHERE subscriber_id = $1 
		AND category = $2 
		AND target_uuid = $3
	`

	result, err := s.DB.Exec(query, subscriberID, category, targetUUID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("subscription not found")
	}

	log.Printf("Removed subscription for subscriber %d, category %s, target %s",
		subscriberID, category, targetUUID)
	return nil
}

// IsSubscribed checks if a subscriber is subscribed to a specific target
func (s *SubscriberService) IsSubscribed(subscriberID int, category models.SubscriptionCategory, targetUUID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM subscriptions 
			WHERE subscriber_id = $1 
			AND category = $2 
			AND target_uuid = $3
		)
	`

	var exists bool
	err := s.DB.QueryRow(query, subscriberID, category, targetUUID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// GetSubscriptionsForSubscriber retrieves all subscriptions for a subscriber
func (s *SubscriberService) GetSubscriptionsForSubscriber(subscriberID int) ([]models.Subscription, error) {
	query := `
		SELECT subscription_id, subscriber_id, category, target_uuid, subscribed_at
		FROM subscriptions 
		WHERE subscriber_id = $1
		ORDER BY subscribed_at DESC
	`

	rows, err := s.DB.Query(query, subscriberID)
	if err != nil {
		return nil, fmt.Errorf("error querying subscriptions: %w", err)
	}
	defer rows.Close()

	var subscriptions []models.Subscription
	for rows.Next() {
		var subscription models.Subscription
		err := rows.Scan(
			&subscription.SubscriptionID,
			&subscription.SubscriberID,
			&subscription.Category,
			&subscription.TargetUUID,
			&subscription.SubscribedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning subscription: %w", err)
		}

		subscriptions = append(subscriptions, subscription)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating subscriptions: %w", err)
	}

	return subscriptions, nil
}

// This method was already defined in subscriber_service.go
// The original GetEventSubscribers method will be used
