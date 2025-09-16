package config

import (
	"log"
	"os"
)

type Config struct {
	AWSRegion          string
	AWSEndpoint        string
	EventServiceURL    string
	KeycloakURL        string
	KeycloakRealm      string
	ClientID           string
	ClientSecret       string
	SQSONSaleQueueURL  string
	SQSSClosedQueueURL string
}

func Load() Config {
	log.Println("Loading configuration from environment variables")
	return Config{
		SQSONSaleQueueURL:  getEnv("AWS_SQS_SESSION_ON_SALE_URL", ""),
		SQSSClosedQueueURL: getEnv("AWS_SQS_SESSION_CLOSED_URL", ""),
		AWSRegion:          getEnv("AWS_REGION", "ap-south-1"),
		AWSEndpoint:        getEnv("AWS_LOCAL_ENDPOINT_URL", ""),
		EventServiceURL:    getEnv("EVENT_SERVICE_URL", "http://localhost:8081/api/event-seating"),
		KeycloakURL:        getEnv("KEYCLOAK_URL", "http://auth.ticketly.com:8080"),
		KeycloakRealm:      getEnv("KEYCLOAK_REALM", "event-ticketing"),
		ClientID:           getEnv("KEYCLOAK_CLIENT_ID", "scheduler-service-client"),
		ClientSecret:       getEnv("SCHEDULER_CLIENT_SECRET", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		log.Printf("Loaded env var %s: %s", key, value)
		return value
	}
	log.Printf("Env var %s not set, using fallback: %s", key, fallback)
	return fallback
}
