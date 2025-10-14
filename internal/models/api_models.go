package models

import (
	"time"
)

// SessionExtendedInfo represents the response from the event-query service's extended session info endpoint
type SessionExtendedInfo struct {
	SessionID      string      `json:"sessionId"`
	EventID        string      `json:"eventId"`
	EventTitle     string      `json:"eventTitle"`
	StartTime      time.Time   `json:"startTime"`
	EndTime        time.Time   `json:"endTime"`
	SalesStartTime time.Time   `json:"salesStartTime"`
	Status         string      `json:"status"`
	SessionType    string      `json:"sessionType"`
	VenueDetails   VenueDetail `json:"venueDetails"`
}

// VenueDetail represents the venue details from the session extended info
type VenueDetail struct {
	Name       string      `json:"name"`
	Address    string      `json:"address"`
	OnlineLink string      `json:"onlineLink"`
	Location   GeoLocation `json:"location"`
}

// GeoLocation represents geographic coordinates
type GeoLocation struct {
	X           float64   `json:"x"`
	Y           float64   `json:"y"`
	Coordinates []float64 `json:"coordinates"`
	Type        string    `json:"type"`
}

// EventBasicInfo represents the response from the event-query service's basic event info endpoint
type EventBasicInfo struct {
	ID           string           `json:"id"`
	Title        string           `json:"title"`
	Description  string           `json:"description"`
	Overview     string           `json:"overview"`
	CoverPhotos  []string         `json:"coverPhotos"`
	Organization OrganizationInfo `json:"organization"`
	Category     CategoryInfo     `json:"category"`
	Tiers        []TierInfo       `json:"tiers"`
}

// OrganizationInfo represents organization information
type OrganizationInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	LogoURL string `json:"logoUrl"`
}

// CategoryInfo represents event category information
type CategoryInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ParentName string `json:"parentName"`
}

// TierInfo represents ticket tier information
type TierInfo struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Color string  `json:"color"`
}
