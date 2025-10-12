package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	AWSRegion                    string
	AWSEndpoint                  string
	EventServiceURL              string
	EventQueryServiceURL         string
	KeycloakURL                  string
	KeycloakRealm                string
	ClientID                     string
	ClientSecret                 string
	KafkaURL                     string
	EventSessionsKafkaTopic      string
	SQSSessionSchedulingQueueURL string
	SQSSessionSchedulingQueueARN string
	SQSTrendingQueueURL          string
	SQSTrendingQueueARN          string
	SchedulerRoleARN             string
	SchedulerGroupName           string
}

// LoadEnv loads environment variables from .env files
func LoadEnv() {
	// Try to find the .env file from the current working directory
	// and from the directory where the binary is located
	envPaths := []string{
		".env",    // Current directory
		"../.env", // One level up
		filepath.Join(os.Getenv("HOME"), "projects/ticketly/ms-scheduling/.env"), // Specific project path
	}

	for _, path := range envPaths {
		err := godotenv.Load(path)
		if err == nil {
			log.Printf("Loaded environment variables from %s", path)
			return
		}
	}

	log.Println("No .env file found, using environment variables")
}

func Load() Config {
	// Load environment variables from .env file first
	LoadEnv()

	log.Println("Loading configuration from environment variables")
	return Config{
		AWSRegion:                    getEnv("AWS_REGION", "ap-south-1"),
		AWSEndpoint:                  getEnv("AWS_LOCAL_ENDPOINT_URL", ""),
		EventServiceURL:              getEnv("EVENT_SERVICE_URL", "http://localhost:8081/api/event-seating"),
		EventQueryServiceURL:         getEnv("EVENT_QUERY_SERVICE_URL", "http://localhost:8082/api/event-query"),
		KeycloakURL:                  getEnv("KEYCLOAK_URL", "http://auth.ticketly.com:8080"),
		KeycloakRealm:                getEnv("KEYCLOAK_REALM", "event-ticketing"),
		ClientID:                     getEnv("KEYCLOAK_CLIENT_ID", "scheduler-service-client"),
		ClientSecret:                 getEnv("SCHEDULER_CLIENT_SECRET", ""),
		KafkaURL:                     getEnv("KAFKA_URL", "localhost:9092"),
		EventSessionsKafkaTopic:      getEnv("EVENT_SESSIONS_KAFKA_TOPIC", "dbz.ticketly.public.event_sessions"),
		SQSSessionSchedulingQueueURL: getEnv("AWS_SQS_SESSION_SCHEDULING_URL", ""),
		SQSSessionSchedulingQueueARN: getEnv("AWS_SQS_SESSION_SCHEDULING_ARN", ""),
		SQSTrendingQueueURL:          getEnv("AWS_SQS_TRENDING_JOB_URL", ""),
		SQSTrendingQueueARN:          getEnv("AWS_SQS_TRENDING_JOB_ARN", ""),
		SchedulerRoleARN:             getEnv("AWS_SCHEDULER_ROLE_ARN", ""),
		SchedulerGroupName:           getEnv("AWS_SCHEDULER_GROUP_NAME", "default"),
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
