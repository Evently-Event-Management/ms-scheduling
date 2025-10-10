package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// SubscriptionCategory represents the subscription category enum
type SubscriptionCategory string

const (
	SubscriptionCategoryOrganization SubscriptionCategory = "organization"
	SubscriptionCategoryEvent        SubscriptionCategory = "event"
	SubscriptionCategorySession      SubscriptionCategory = "session"
)

// Scan implements the sql.Scanner interface for SubscriptionCategory
func (sc *SubscriptionCategory) Scan(value interface{}) error {
	if value == nil {
		*sc = ""
		return nil
	}
	if str, ok := value.(string); ok {
		*sc = SubscriptionCategory(str)
		return nil
	}
	return fmt.Errorf("cannot scan %T into SubscriptionCategory", value)
}

// Value implements the driver.Valuer interface for SubscriptionCategory
func (sc SubscriptionCategory) Value() (driver.Value, error) {
	return string(sc), nil
}

// Subscriber represents a subscriber in the system
type Subscriber struct {
	SubscriberID   int       `json:"subscriber_id" db:"subscriber_id"`
	UserID         *string   `json:"user_id,omitempty" db:"user_id"` // Keycloak UUID
	SubscriberMail string    `json:"subscriber_mail" db:"subscriber_mail"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// Subscription represents a subscription record
type Subscription struct {
	SubscriptionID int                  `json:"subscription_id" db:"subscription_id"`
	SubscriberID   int                  `json:"subscriber_id" db:"subscriber_id"`
	Category       SubscriptionCategory `json:"category" db:"category"`
	TargetID       int                  `json:"target_id" db:"target_id"`
	SubscribedAt   time.Time            `json:"subscribed_at" db:"subscribed_at"`
}

// SubscriptionRequest represents a request to create a subscription
type SubscriptionRequest struct {
	SubscriberMail string               `json:"subscriber_mail" validate:"required,email"`
	Category       SubscriptionCategory `json:"category" validate:"required"`
	TargetID       int                  `json:"target_id" validate:"required"`
}
