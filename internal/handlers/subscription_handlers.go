package handlers

import (
	"encoding/json"
	"log"
	"ms-scheduling/internal/auth"
	"ms-scheduling/internal/config"
	"ms-scheduling/internal/models"
	"ms-scheduling/internal/services"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type SubscriptionHandler struct {
	subscriberService *services.SubscriberService
	cfg               config.Config
}

func NewSubscriptionHandler(subscriberService *services.SubscriberService, cfg config.Config) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriberService: subscriberService,
		cfg:               cfg,
	}
}

// Subscribe handles POST /subscription/v1/subscribe
func (h *SubscriptionHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from token
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		log.Printf("Error getting user ID from context: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var subscribeRequest struct {
		EventID string `json:"eventId"`
	}

	err = json.NewDecoder(r.Body).Decode(&subscribeRequest)
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if subscribeRequest.EventID == "" {
		http.Error(w, "EventID is required", http.StatusBadRequest)
		return
	}

	// Get or create subscriber
	subscriber, err := h.subscriberService.GetOrCreateSubscriber(userID)
	if err != nil {
		log.Printf("Error getting/creating subscriber: %v", err)
		http.Error(w, "Failed to process subscription", http.StatusInternalServerError)
		return
	}

	// Add subscription
	err = h.subscriberService.AddSubscription(subscriber.SubscriberID, models.SubscriptionCategoryEvent, subscribeRequest.EventID)
	if err != nil {
		log.Printf("Error adding subscription: %v", err)
		http.Error(w, "Failed to create subscription", http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Subscription created successfully",
		"eventId": subscribeRequest.EventID,
	})
}

// Unsubscribe handles DELETE /subscription/v1/unsubscribe/:eventId
func (h *SubscriptionHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from token
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		log.Printf("Error getting user ID from context: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event ID from URL path
	vars := mux.Vars(r)
	eventID := vars["eventId"]
	if eventID == "" {
		http.Error(w, "EventID is required", http.StatusBadRequest)
		return
	}

	// Get subscriber
	subscriber, err := h.subscriberService.GetOrCreateSubscriber(userID)
	if err != nil {
		log.Printf("Error getting subscriber: %v", err)
		http.Error(w, "Failed to process unsubscription", http.StatusInternalServerError)
		return
	}

	// Remove subscription
	err = h.subscriberService.RemoveSubscription(subscriber.SubscriberID, models.SubscriptionCategoryEvent, eventID)
	if err != nil {
		log.Printf("Error removing subscription: %v", err)
		http.Error(w, "Failed to remove subscription", http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Unsubscribed successfully",
		"eventId": eventID,
	})
}

// IsSubscribed handles GET /subscription/v1/is-subscribed/:eventId
func (h *SubscriptionHandler) IsSubscribed(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from token
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		log.Printf("Error getting user ID from context: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event ID from URL path
	vars := mux.Vars(r)
	eventID := vars["eventId"]
	if eventID == "" {
		http.Error(w, "EventID is required", http.StatusBadRequest)
		return
	}

	// Get subscriber
	subscriber, err := h.subscriberService.GetOrCreateSubscriber(userID)
	if err != nil {
		log.Printf("Error getting subscriber: %v", err)
		http.Error(w, "Failed to check subscription", http.StatusInternalServerError)
		return
	}

	// Check subscription
	isSubscribed, err := h.subscriberService.IsSubscribed(subscriber.SubscriberID, models.SubscriptionCategoryEvent, eventID)
	if err != nil {
		log.Printf("Error checking subscription: %v", err)
		http.Error(w, "Failed to check subscription", http.StatusInternalServerError)
		return
	}

	// Return result
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"isSubscribed": isSubscribed,
		"eventId":      eventID,
	})
}

// GetUserSubscriptions handles GET /subscription/v1/user-subscriptions
func (h *SubscriptionHandler) GetUserSubscriptions(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from token
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		log.Printf("Error getting user ID from context: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get subscriber
	subscriber, err := h.subscriberService.GetOrCreateSubscriber(userID)
	if err != nil {
		log.Printf("Error getting subscriber: %v", err)
		http.Error(w, "Failed to get subscriptions", http.StatusInternalServerError)
		return
	}

	// Get subscriptions
	subscriptions, err := h.subscriberService.GetSubscriptionsForSubscriber(subscriber.SubscriberID)
	if err != nil {
		log.Printf("Error getting subscriptions: %v", err)
		http.Error(w, "Failed to get subscriptions", http.StatusInternalServerError)
		return
	}

	// Return result
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"subscriptions": subscriptions,
	})
}

// GetEventSubscribers handles GET /subscription/v1/event-subscribers/:eventId
func (h *SubscriptionHandler) GetEventSubscribers(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	isAdmin, err := h.isUserAdmin(r)
	if err != nil || !isAdmin {
		log.Printf("User is not authorized to access this endpoint: %v", err)
		http.Error(w, "Unauthorized - Admin access required", http.StatusForbidden)
		return
	}

	// Get event ID from URL path
	vars := mux.Vars(r)
	eventID := vars["eventId"]
	if eventID == "" {
		http.Error(w, "EventID is required", http.StatusBadRequest)
		return
	}

	// Pagination parameters
	page := 1
	pageSize := 20

	// Parse query parameters
	pageParam := r.URL.Query().Get("page")
	if pageParam != "" {
		pageInt, err := strconv.Atoi(pageParam)
		if err == nil && pageInt > 0 {
			page = pageInt
		}
	}

	pageSizeParam := r.URL.Query().Get("pageSize")
	if pageSizeParam != "" {
		pageSizeInt, err := strconv.Atoi(pageSizeParam)
		if err == nil && pageSizeInt > 0 && pageSizeInt <= 100 {
			pageSize = pageSizeInt
		}
	}

	// Get subscribers
	subscribers, err := h.subscriberService.GetEventSubscribers(eventID)
	if err != nil {
		log.Printf("Error getting event subscribers: %v", err)
		http.Error(w, "Failed to get subscribers", http.StatusInternalServerError)
		return
	}

	// For simple implementation, we'll do manual pagination in memory
	totalCount := len(subscribers)

	// Calculate pagination info
	totalPages := (totalCount + pageSize - 1) / pageSize
	hasNext := page < totalPages
	hasPrev := page > 1

	// Apply pagination manually
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(subscribers) {
		// Return empty list if start is beyond the available data
		subscribers = []models.Subscriber{}
	} else if end > len(subscribers) {
		// If end is beyond the available data, limit to available data
		subscribers = subscribers[start:]
	} else {
		subscribers = subscribers[start:end]
	}

	// Return result
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"subscribers": subscribers,
		"pagination": map[string]interface{}{
			"page":       page,
			"pageSize":   pageSize,
			"totalCount": totalCount,
			"totalPages": totalPages,
			"hasNext":    hasNext,
			"hasPrev":    hasPrev,
		},
	})
}

// isUserAdmin checks if the user has admin role in their token
func (h *SubscriptionHandler) isUserAdmin(r *http.Request) (bool, error) {
	// Get the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false, nil
	}

	// Extract the token
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return false, nil
	}
	token := parts[1]

	// Check if token has admin role
	// In a real implementation, this would verify the JWT and check for admin role
	// For now, we'll use a simple check based on token claims
	return auth.HasAdminRole(token)
}
