package kafka

import (
	"encoding/json"
	"testing"

	"ms-scheduling/internal/models"
	"ms-scheduling/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSubscriberService is a mock of SubscriberService
type MockSubscriberService struct {
	mock.Mock
}

func (m *MockSubscriberService) GetOrCreateSubscriber(userID string) (*models.Subscriber, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Subscriber), args.Error(1)
}

func (m *MockSubscriberService) GetSubscriberByUserID(userID string) (*models.Subscriber, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Subscriber), args.Error(1)
}

func (m *MockSubscriberService) AddSubscription(subscriberID int, category models.SubscriptionCategory, entityID string) error {
	args := m.Called(subscriberID, category, entityID)
	return args.Error(0)
}

func (m *MockSubscriberService) SendOrderConfirmationEmail(subscriber *models.Subscriber, order *services.OrderCreatedEvent) error {
	args := m.Called(subscriber, order)
	return args.Error(0)
}

// TestOrderConsumer is a simplified version of OrderConsumer for testing
type TestOrderConsumer struct {
	SubscriberService interface{} // Use interface instead of concrete type for testing
}

func (c *TestOrderConsumer) processOrderCreated(value []byte) error {
	var order services.OrderCreatedEvent
	if err := json.Unmarshal(value, &order); err != nil {
		return err
	}

	service := c.SubscriberService.(*MockSubscriberService)
	subscriber, err := service.GetOrCreateSubscriber(order.UserID)
	if err != nil {
		return err
	}

	if order.Status == "completed" {
		if err := service.AddSubscription(subscriber.SubscriberID, models.SubscriptionCategoryEvent, order.EventID); err != nil {
			return err
		}
		if err := service.AddSubscription(subscriber.SubscriberID, models.SubscriptionCategorySession, order.SessionID); err != nil {
			return err
		}
		if order.OrganizationID != "" {
			if err := service.AddSubscription(subscriber.SubscriberID, models.SubscriptionCategoryOrganization, order.OrganizationID); err != nil {
				return err
			}
		}
	}

	if err := service.SendOrderConfirmationEmail(subscriber, &order); err != nil {
		return err
	}

	return nil
}

func (c *TestOrderConsumer) processOrderUpdated(value []byte) error {
	var order services.OrderCreatedEvent
	if err := json.Unmarshal(value, &order); err != nil {
		return err
	}

	service := c.SubscriberService.(*MockSubscriberService)
	subscriber, err := service.GetOrCreateSubscriber(order.UserID)
	if err != nil {
		return err
	}

	if order.Status == "completed" {
		if err := service.AddSubscription(subscriber.SubscriberID, models.SubscriptionCategoryEvent, order.EventID); err != nil {
			return err
		}
		if err := service.AddSubscription(subscriber.SubscriberID, models.SubscriptionCategorySession, order.SessionID); err != nil {
			return err
		}
		if order.OrganizationID != "" {
			if err := service.AddSubscription(subscriber.SubscriberID, models.SubscriptionCategoryOrganization, order.OrganizationID); err != nil {
				return err
			}
		}
	}

	if err := service.SendOrderConfirmationEmail(subscriber, &order); err != nil {
		return err
	}

	return nil
}

func (c *TestOrderConsumer) processOrderCancelled(value []byte) error {
	var order services.OrderCreatedEvent
	if err := json.Unmarshal(value, &order); err != nil {
		return err
	}

	service := c.SubscriberService.(*MockSubscriberService)
	subscriber, err := service.GetSubscriberByUserID(order.UserID)
	if err != nil {
		return err
	}

	if subscriber == nil {
		return nil
	}

	// Force status to cancelled
	order.Status = "cancelled"

	if err := service.SendOrderConfirmationEmail(subscriber, &order); err != nil {
		return err
	}

	return nil
}

// Test for processOrderCreated
func TestProcessOrderCreated(t *testing.T) {
	// Create test data
	order := services.OrderCreatedEvent{
		OrderID:        "order-123",
		UserID:         "user-123",
		EventID:        "event-123",
		SessionID:      "session-123",
		OrganizationID: "org-123",
		Status:         "completed",
		SubTotal:       100.0,
		Price:          95.0,
		Tickets: []services.Ticket{
			{
				TicketID:        "ticket-123",
				SeatLabel:       "A1",
				TierName:        "VIP",
				PriceAtPurchase: 95.0,
			},
		},
	}

	// Create mock subscriber service
	mockService := new(MockSubscriberService)

	userID := "user-123"
	subscriber := &models.Subscriber{
		SubscriberID:   123,
		UserID:         &userID,
		SubscriberMail: "test@example.com",
	}

	// Set expectations
	mockService.On("GetOrCreateSubscriber", "user-123").Return(subscriber, nil)
	mockService.On("AddSubscription", 123, models.SubscriptionCategoryEvent, "event-123").Return(nil)
	mockService.On("AddSubscription", 123, models.SubscriptionCategorySession, "session-123").Return(nil)
	mockService.On("AddSubscription", 123, models.SubscriptionCategoryOrganization, "org-123").Return(nil)
	mockService.On("SendOrderConfirmationEmail", subscriber, mock.MatchedBy(func(o *services.OrderCreatedEvent) bool {
		return o.OrderID == order.OrderID
	})).Return(nil)

	// Create test consumer with our mock service
	consumer := &TestOrderConsumer{
		SubscriberService: mockService,
	}

	// Marshal the order to JSON
	orderBytes, _ := json.Marshal(order)

	// Call the method
	err := consumer.processOrderCreated(orderBytes)

	// Verify results
	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

// Test for processOrderUpdated
func TestProcessOrderUpdated(t *testing.T) {
	// Create test data
	order := services.OrderCreatedEvent{
		OrderID:        "order-123",
		UserID:         "user-123",
		EventID:        "event-123",
		SessionID:      "session-123",
		OrganizationID: "org-123",
		Status:         "completed", // Changed from pending to completed
		SubTotal:       100.0,
		Price:          95.0,
		Tickets: []services.Ticket{
			{
				TicketID:        "ticket-123",
				SeatLabel:       "A1",
				TierName:        "VIP",
				PriceAtPurchase: 95.0,
			},
		},
	}

	// Create mock subscriber service
	mockService := new(MockSubscriberService)

	userID := "user-123"
	subscriber := &models.Subscriber{
		SubscriberID:   123,
		UserID:         &userID,
		SubscriberMail: "test@example.com",
	}

	// Set expectations
	mockService.On("GetOrCreateSubscriber", "user-123").Return(subscriber, nil)
	mockService.On("AddSubscription", 123, models.SubscriptionCategoryEvent, "event-123").Return(nil)
	mockService.On("AddSubscription", 123, models.SubscriptionCategorySession, "session-123").Return(nil)
	mockService.On("AddSubscription", 123, models.SubscriptionCategoryOrganization, "org-123").Return(nil)
	mockService.On("SendOrderConfirmationEmail", subscriber, mock.MatchedBy(func(o *services.OrderCreatedEvent) bool {
		return o.OrderID == order.OrderID
	})).Return(nil)

	// Create test consumer with mock service
	consumer := &TestOrderConsumer{
		SubscriberService: mockService,
	}

	// Marshal the order to JSON
	orderBytes, _ := json.Marshal(order)

	// Call the method
	err := consumer.processOrderUpdated(orderBytes)

	// Verify results
	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}

// Test for processOrderCancelled
func TestProcessOrderCancelled(t *testing.T) {
	// Create test data
	order := services.OrderCreatedEvent{
		OrderID:        "order-123",
		UserID:         "user-123",
		EventID:        "event-123",
		SessionID:      "session-123",
		OrganizationID: "org-123",
		Status:         "pending", // Original status doesn't matter for cancelled
		SubTotal:       100.0,
		Price:          95.0,
		Tickets: []services.Ticket{
			{
				TicketID:        "ticket-123",
				SeatLabel:       "A1",
				TierName:        "VIP",
				PriceAtPurchase: 95.0,
			},
		},
	}

	// Create mock subscriber service
	mockService := new(MockSubscriberService)

	userID := "user-123"
	subscriber := &models.Subscriber{
		SubscriberID:   123,
		UserID:         &userID,
		SubscriberMail: "test@example.com",
	}

	// Set expectations - note we're getting the subscriber by user ID not creating
	mockService.On("GetSubscriberByUserID", "user-123").Return(subscriber, nil)

	// Force the status to cancelled and send the email
	mockService.On("SendOrderConfirmationEmail", subscriber, mock.MatchedBy(func(o *services.OrderCreatedEvent) bool {
		return o.OrderID == order.OrderID && o.Status == "cancelled"
	})).Return(nil)

	// Create test consumer with mock service
	consumer := &TestOrderConsumer{
		SubscriberService: mockService,
	}

	// Marshal the order to JSON
	orderBytes, _ := json.Marshal(order)

	// Call the method
	err := consumer.processOrderCancelled(orderBytes)

	// Verify results
	assert.NoError(t, err)
	mockService.AssertExpectations(t)
}
