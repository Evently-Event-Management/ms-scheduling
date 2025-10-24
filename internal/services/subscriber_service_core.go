package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"ms-scheduling/internal/config"
	"ms-scheduling/internal/email"
	"net/http"
)

type SubscriberService struct {
	DB             *sql.DB
	KeycloakClient *KeycloakClient
	EmailService   *EmailService
	EmailManager   *email.EmailManager
	Config         *config.Config
}

func NewSubscriberService(db *sql.DB, keycloakClient *KeycloakClient, emailService *EmailService, cfg *config.Config) *SubscriberService {
	return &SubscriberService{
		DB:             db,
		KeycloakClient: keycloakClient,
		EmailService:   emailService,
		EmailManager:   nil, // Will be set later to avoid circular dependencies
		Config:         cfg,
	}
}

// SetEmailManager sets the email manager (used to avoid circular dependencies)
func (s *SubscriberService) SetEmailManager(emailManager *email.EmailManager) {
	s.EmailManager = emailManager
}

// getEventTitle fetches the event title from the database
func (s *SubscriberService) getEventTitle(eventID string) string {
	// Note: events table may not exist in this service's database
	// This is a cross-service reference - event data is in event-service
	// For now, we'll use the event ID as fallback
	var title string
	query := `SELECT title FROM events WHERE id = $1`
	err := s.DB.QueryRow(query, eventID).Scan(&title)
	if err != nil {
		// Don't log error - events table doesn't exist in this microservice
		// Return a user-friendly fallback
		return "Event " + eventID[:8] + "..." // Show first 8 chars of UUID
	}
	return title
}

// getOrganizationName fetches the organization name from the database
func (s *SubscriberService) getOrganizationName(organizationID string) string {
	// Note: organizations table may not exist in this service's database
	// This is a cross-service reference - organization data is in org-service
	// For now, we'll use the organization ID as fallback
	var name string
	query := `SELECT name FROM organizations WHERE id = $1`
	err := s.DB.QueryRow(query, organizationID).Scan(&name)
	if err != nil {
		// Don't log error - organizations table doesn't exist in this microservice
		// Return a user-friendly fallback
		return "Organization " + organizationID[:8] + "..." // Show first 8 chars of UUID
	}
	return name
}

// EventBasicInfo represents the event information from event-query service
type EventBasicInfo struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Overview     string   `json:"overview"`
	CoverPhotos  []string `json:"coverPhotos"`
	Organization struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		LogoURL string `json:"logoUrl"`
	} `json:"organization"`
	Category struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		ParentName string `json:"parentName"`
	} `json:"category"`
	Tiers []struct {
		ID    string  `json:"id"`
		Name  string  `json:"name"`
		Price float64 `json:"price"`
		Color string  `json:"color"`
	} `json:"tiers"`
}

// getEventBasicInfo fetches event details from event-query service
func (s *SubscriberService) getEventBasicInfo(eventID string) (*EventBasicInfo, error) {
	// Use the event-query service URL from config or default
	baseURL := "http://localhost:8088" // TODO: Move to config
	if s.Config != nil && s.Config.EventQueryServiceURL != "" {
		baseURL = s.Config.EventQueryServiceURL
	}

	url := fmt.Sprintf("%s/api/event-query/v1/events/%s/basic-info", baseURL, eventID)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching event basic info for %s: %v", eventID, err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Event query service returned status %d for event %s: %s", resp.StatusCode, eventID, string(body))
		return nil, fmt.Errorf("event query service returned status %d", resp.StatusCode)
	}

	var eventInfo EventBasicInfo
	if err := json.NewDecoder(resp.Body).Decode(&eventInfo); err != nil {
		log.Printf("Error decoding event basic info for %s: %v", eventID, err)
		return nil, err
	}

	return &eventInfo, nil
}
