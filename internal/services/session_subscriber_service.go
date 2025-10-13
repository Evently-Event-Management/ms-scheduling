package services

import (
	"fmt"
	"log"
	"ms-scheduling/internal/models"
)

// GetSessionSubscriptionsForSubscriber retrieves all session subscriptions for a subscriber
func (s *SubscriberService) GetSessionSubscriptionsForSubscriber(subscriberID int) ([]models.Subscription, error) {
	query := `
		SELECT subscription_id, subscriber_id, category, target_uuid, subscribed_at
		FROM subscriptions 
		WHERE subscriber_id = $1 AND category = $2
		ORDER BY subscribed_at DESC
	`

	rows, err := s.DB.Query(query, subscriberID, models.SubscriptionCategorySession)
	if err != nil {
		return nil, fmt.Errorf("error querying session subscriptions: %w", err)
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

	log.Printf("Found %d session subscriptions for subscriber %d", len(subscriptions), subscriberID)
	return subscriptions, nil
}