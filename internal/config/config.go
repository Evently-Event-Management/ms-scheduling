package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	AWSRegion                    string
	AWSEndpoint                  string
	AWSAccessKeyID               string
	AWSSecretAccessKey           string
	EventServiceURL              string
	EventQueryServiceURL         string
	KeycloakURL                  string
	KeycloakRealm                string
	ClientID                     string
	ClientSecret                 string
	KafkaURL                     string
	EventSessionsKafkaTopic      string
	OrdersKafkaTopic             string
	OrdersUpdatedKafkaTopic      string
	OrdersCancelledKafkaTopic    string
	EventsKafkaTopic             string
	FrontendURL                  string
	SQSSessionSchedulingQueueURL string
	SQSSessionSchedulingQueueARN string
	SQSSessionRemindersQueueURL  string
	SQSSessionRemindersQueueARN  string
	SQSTrendingQueueURL          string
	SQSTrendingQueueARN          string
	SchedulerRoleARN             string
	SchedulerGroupName           string

	// Database configuration
	PostgresDSN string

	// Email configuration
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string

	// HTTP server configuration
	ServerHost string
	ServerPort string
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
		AWSAccessKeyID:               getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey:           getEnv("AWS_SECRET_ACCESS_KEY", ""),
		EventServiceURL:              getEnv("EVENT_SERVICE_URL", "http://localhost:8081/api/event-seating"),
		EventQueryServiceURL:         getEnv("EVENT_QUERY_SERVICE_URL", "http://localhost:8082/api/event-query"),
		KeycloakURL:                  getEnv("KEYCLOAK_URL", "http://auth.ticketly.com:8080"),
		KeycloakRealm:                getEnv("KEYCLOAK_REALM", "event-ticketing"),
		ClientID:                     getEnv("KEYCLOAK_CLIENT_ID", "scheduler-service-client"),
		ClientSecret:                 getEnv("SCHEDULER_CLIENT_SECRET", ""),
		KafkaURL:                     getEnv("KAFKA_URL", "localhost:9092"),
		SQSSessionSchedulingQueueURL: getEnv("AWS_SQS_SESSION_SCHEDULING_URL", ""),
		SQSSessionSchedulingQueueARN: getEnv("AWS_SQS_SESSION_SCHEDULING_ARN", ""),
		SQSSessionRemindersQueueURL:  getEnv("AWS_SQS_SESSION_REMINDERS_URL", ""),
		SQSSessionRemindersQueueARN:  getEnv("AWS_SQS_SESSION_REMINDERS_ARN", ""),
		SQSTrendingQueueURL:          getEnv("AWS_SQS_TRENDING_JOB_URL", ""),
		SQSTrendingQueueARN:          getEnv("AWS_SQS_TRENDING_JOB_ARN", ""),
		SchedulerRoleARN:             getEnv("AWS_SCHEDULER_ROLE_ARN", ""),
		SchedulerGroupName:           getEnv("AWS_SCHEDULER_GROUP_NAME", "default"),
		EventSessionsKafkaTopic:      getEnv("EVENT_SESSIONS_KAFKA_TOPIC", "dbz.ticketly.public.event_sessions"),
		OrdersKafkaTopic:             getEnv("ORDERS_KAFKA_TOPIC", "ticketly.order.created"),
		OrdersUpdatedKafkaTopic:      getEnv("ORDERS_UPDATED_KAFKA_TOPIC", "ticketly.order.updated"),
		OrdersCancelledKafkaTopic:    getEnv("ORDERS_CANCELLED_KAFKA_TOPIC", "ticketly.order.cancelled"),
		EventsKafkaTopic:             getEnv("EVENTS_KAFKA_TOPIC", "dbz.ticketly.public.events"),
		FrontendURL:                  getEnv("FRONTEND_URL", "https://ticketly.dpiyumal.me"),

		// Database configuration
		PostgresDSN: getEnv("POSTGRES_DSN", "host=localhost port=5432 user=postgres password= dbname=ticketly sslmode=disable"),

		// Email configuration
		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		FromEmail:    getEnv("FROM_EMAIL", "noreply@ticketly.com"),
		FromName:     getEnv("FROM_NAME", "Ticketly"),

		// HTTP server configuration
		ServerHost: getEnv("SERVER_HOST", "0.0.0.0"),
		ServerPort: getEnv("SERVER_PORT", "8085"),
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
