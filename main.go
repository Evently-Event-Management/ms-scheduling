package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"sync"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awsscheduler "github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	auth "ms-scheduling/internal/auth"
	appconfig "ms-scheduling/internal/config"
	"ms-scheduling/internal/eventbridge"
	"ms-scheduling/internal/kafka"
	"ms-scheduling/internal/reminder"
	"ms-scheduling/internal/scheduler"
	"ms-scheduling/internal/services"
	"ms-scheduling/internal/trending"
)

// Types moved to internal packages.

// Main application loop
func main() {
	// Parse command line flags
	testUserID := flag.String("test-user", "", "Test getting email for a specific user ID")
	flag.Parse()

	cfg := appconfig.Load()
	log.Printf("Loaded config: %+v", cfg)

	// Create clients once, outside the loop
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// If a user ID is provided, test the GetUserEmailByID function
	if *testUserID != "" {
		testGetUserEmail(cfg, httpClient, *testUserID)
		return
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		log.Fatalf("unable to load AWS SDK config, %v", err)
	}
	sqsClient := sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		if cfg.AWSEndpoint != "" {
			log.Printf("Using LocalStack endpoint for AWS services: %s", cfg.AWSEndpoint)
			o.BaseEndpoint = &cfg.AWSEndpoint
		}
	})
	log.Println("Clients initialized")

	schedulerClient := awsscheduler.NewFromConfig(awsCfg)

	// Initialize the scheduler service
	schedulerService := eventbridge.NewService(cfg, schedulerClient)

	// Initialize database service
	dbConfig := services.DatabaseConfig{
		Host:     cfg.DatabaseHost,
		Port:     cfg.DatabasePort,
		User:     cfg.DatabaseUser,
		Password: cfg.DatabasePassword,
		DBName:   cfg.DatabaseName,
		SSLMode:  cfg.DatabaseSSLMode,
	}
	dbService, err := services.NewDatabaseService(dbConfig)
	if err != nil {
		log.Fatalf("Failed to initialize database service: %v", err)
	}
	defer dbService.Close()

	// Initialize database tables
	if err := dbService.InitializeTables(); err != nil {
		log.Fatalf("Failed to initialize database tables: %v", err)
	}

	// Initialize Keycloak client
	keycloakClient := services.NewKeycloakClient(cfg.KeycloakURL, cfg.KeycloakRealm, cfg.ClientID, cfg.ClientSecret)

	// Initialize email service
	emailService := services.NewEmailService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.FromEmail, cfg.FromName)

	// Initialize subscriber service
	subscriberService := services.NewSubscriberService(dbService.DB, keycloakClient, emailService)

	// Start Kafka consumer in a separate goroutine if Kafka URL is configured
	if cfg.KafkaURL != "" && cfg.EventSessionsKafkaTopic != "" {
		log.Printf("Starting Kafka consumer for topic %s at %s", cfg.EventSessionsKafkaTopic, cfg.KafkaURL)
		kafkaConsumer := kafka.NewConsumer(cfg, cfg.KafkaURL, cfg.EventSessionsKafkaTopic, schedulerService, subscriberService)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			kafkaConsumer.ConsumeDebeziumEvents()
		}()
		// We don't wait for wg.Wait() so the SQS processing can continue
	} else {
		log.Println("Kafka URL or topic not configured, skipping Kafka consumer setup")
	}

	// Start trending job processor in a separate goroutine if trending queue URL is configured
	if cfg.SQSTrendingQueueURL != "" {
		log.Printf("Starting trending job processor for queue: %s", cfg.SQSTrendingQueueURL)
		trendingProcessor := trending.NewProcessor(sqsClient, httpClient, cfg)
		var trendingWg sync.WaitGroup
		trendingWg.Add(1)
		go func() {
			defer trendingWg.Done()
			err := trendingProcessor.ProcessMessages(context.Background())
			if err != nil {
				log.Printf("Error processing trending messages: %v", err)
			}
		}()
		// We don't wait for trendingWg.Wait() so other processing can continue
	} else {
		log.Println("Trending queue URL not configured, skipping trending processor setup")
	}

	// Start session scheduling processor in a separate goroutine if session scheduling queue URL is configured
	if cfg.SQSSessionSchedulingQueueURL != "" {
		log.Printf("Starting session scheduling processor for queue: %s", cfg.SQSSessionSchedulingQueueURL)
		sessionProcessor := scheduler.NewProcessor(sqsClient, httpClient, cfg)
		var sessionWg sync.WaitGroup
		sessionWg.Add(1)
		go func() {
			defer sessionWg.Done()
			err := sessionProcessor.ProcessMessages(context.Background())
			if err != nil {
				log.Printf("Error processing session scheduling messages: %v", err)
			}
		}()
		// We don't wait for sessionWg.Wait() so other processing can continue
	} else {
		log.Println("Session scheduling queue URL not configured, skipping session processor setup")
	}

	// Start reminder processor in a separate goroutine if reminder queue URL is configured
	if cfg.SQSSessionRemindersQueueURL != "" {
		log.Printf("Starting reminder processor for queue: %s", cfg.SQSSessionRemindersQueueURL)
		reminderProcessor := reminder.NewProcessor(sqsClient, httpClient, cfg, subscriberService)
		var reminderWg sync.WaitGroup
		reminderWg.Add(1)
		go func() {
			defer reminderWg.Done()
			err := reminderProcessor.ProcessMessages(context.Background())
			if err != nil {
				log.Printf("Error processing reminder messages: %v", err)
			}
		}()
		// We don't wait for reminderWg.Wait() so other processing can continue
	} else {
		log.Println("Reminder queue URL not configured, skipping reminder processor setup")
	}

	// Keep the main goroutine alive
	for {
		time.Sleep(time.Hour)
	}
}

// testGetUserEmail tests the GetUserEmailByID function with the provided user ID
func testGetUserEmail(cfg appconfig.Config, httpClient *http.Client, userID string) {
	log.Printf("Testing GetUserEmailByID with user ID: %s", userID)

	email, err := auth.GetUserEmailByID(cfg, httpClient, userID)
	if err != nil {
		log.Printf("Error getting email for user %s: %v", userID, err)
		return
	}

	log.Printf("Successfully retrieved email for user %s: %s", userID, email)
}
