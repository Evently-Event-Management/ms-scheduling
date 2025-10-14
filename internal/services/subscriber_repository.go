package services

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"ms-scheduling/internal/models"
)

// GetOrCreateSubscriber gets subscriber by user ID or creates a new one
func (s *SubscriberService) GetOrCreateSubscriber(userID string) (*models.Subscriber, error) {
	subscriber, err := s.getSubscriberByUserID(userID)
	if err == nil {
		return subscriber, nil
	}

	email, err := s.KeycloakClient.GetUserEmail(userID)
	if err != nil {
		log.Printf("Warning: Failed to get user email from Keycloak: %v", err)
		email = userID + "@example.com"
	}

	subscriber, err = s.createSubscriber(userID, email)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber: %w", err)
	}

	log.Printf("Created new subscriber for user %s with email %s", userID, email)
	return subscriber, nil
}

// GetSubscriberByUserID returns a subscriber by user ID, or nil if not found
func (s *SubscriberService) GetSubscriberByUserID(userID string) (*models.Subscriber, error) {
	subscriber, err := s.getSubscriberByUserID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No subscriber found, return nil without error
			return nil, nil
		}
		return nil, fmt.Errorf("error getting subscriber by user ID: %w", err)
	}
	return subscriber, nil
}

func (s *SubscriberService) getSubscriberByUserID(userID string) (*models.Subscriber, error) {
	query := `
        SELECT subscriber_id, user_id, subscriber_mail, created_at 
        FROM subscribers 
        WHERE user_id = $1
    `

	var subscriber models.Subscriber
	err := s.DB.QueryRow(query, userID).Scan(
		&subscriber.SubscriberID,
		&subscriber.UserID,
		&subscriber.SubscriberMail,
		&subscriber.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &subscriber, nil
}

func (s *SubscriberService) createSubscriber(userID string, email string) (*models.Subscriber, error) {
	query := `
        INSERT INTO subscribers (user_id, subscriber_mail) 
        VALUES ($1, $2) 
        ON CONFLICT (subscriber_mail) DO UPDATE SET 
            user_id = EXCLUDED.user_id,
            created_at = subscribers.created_at
        RETURNING subscriber_id, user_id, subscriber_mail, created_at
    `

	var subscriber models.Subscriber
	err := s.DB.QueryRow(query, userID, email).Scan(
		&subscriber.SubscriberID,
		&subscriber.UserID,
		&subscriber.SubscriberMail,
		&subscriber.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &subscriber, nil
}

func (s *SubscriberService) AddSubscription(subscriberID int, category models.SubscriptionCategory, targetUUID string) error {
	query := `
        INSERT INTO subscriptions (subscriber_id, category, target_uuid) 
        VALUES ($1, $2, $3) 
        ON CONFLICT (subscriber_id, category, target_uuid) DO NOTHING
    `

	_, err := s.DB.Exec(query, subscriberID, category, targetUUID)
	return err
}
