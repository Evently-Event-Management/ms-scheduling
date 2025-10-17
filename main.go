package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awsscheduler "github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gorilla/mux"

	auth "ms-scheduling/internal/auth"
	"ms-scheduling/internal/config"
	"ms-scheduling/internal/eventbridge"
	"ms-scheduling/internal/handlers"
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

	cfg := config.Load()
	log.Printf("Loaded config: %+v", cfg)

	// Create clients once, outside the loop
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// If a user ID is provided, test the GetUserEmailByID function
	if *testUserID != "" {
		testGetUserEmail(cfg, httpClient, *testUserID)
		return
	}

	// Load AWS configuration with credentials from environment variables
	awsOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.AWSRegion),
	}

	// Add credentials if they are provided
	if cfg.AWSAccessKeyID != "" && cfg.AWSSecretAccessKey != "" {
		log.Println("Using AWS credentials from environment variables")
		awsOptions = append(awsOptions, awsconfig.WithCredentialsProvider(
			aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     cfg.AWSAccessKeyID,
					SecretAccessKey: cfg.AWSSecretAccessKey,
				}, nil
			}),
		))
	} else {
		log.Println("No AWS credentials provided in environment variables, falling back to default credentials")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsOptions...)
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
	dbService, err := services.NewDatabaseService(cfg.PostgresDSN)
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
	subscriberService := services.NewSubscriberService(dbService.DB, keycloakClient, emailService, &cfg)

	// Start Kafka consumers in separate goroutines if Kafka URL is configured
	if cfg.KafkaURL != "" {
		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start event sessions consumer if topic is configured
		if cfg.EventSessionsKafkaTopic != "" {
			log.Printf("Starting event sessions consumer for topic %s at %s", cfg.EventSessionsKafkaTopic, cfg.KafkaURL)
			sessionConsumer := kafka.NewSessionConsumer(cfg, schedulerService, subscriberService)
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := sessionConsumer.StartConsuming(ctx); err != nil {
					log.Printf("Error in session consumer: %v", err)
				}
			}()
		}

		// Start orders consumer if any order topic is configured
		// We'll always create the consumer (it checks for empty topics internally)
		// but only log the actual topics that are configured
		log.Printf("Starting orders consumer for topics (created: %s, updated: %s, cancelled: %s) at %s",
			cfg.OrdersKafkaTopic, cfg.OrdersUpdatedKafkaTopic, cfg.OrdersCancelledKafkaTopic, cfg.KafkaURL)
		orderConsumer := kafka.NewOrderConsumer(cfg, subscriberService)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := orderConsumer.StartConsuming(ctx); err != nil {
				log.Printf("Error in order consumer: %v", err)
			}
		}() // Start events consumer if topic is configured
		if cfg.EventsKafkaTopic != "" {
			log.Printf("Starting events consumer for topic %s at %s", cfg.EventsKafkaTopic, cfg.KafkaURL)
			eventConsumer := kafka.NewEventConsumer(cfg, subscriberService)
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := eventConsumer.StartConsuming(ctx); err != nil {
					log.Printf("Error in event consumer: %v", err)
				}
			}()
		}

		// We don't wait for wg.Wait() so the SQS processing can continue
	} else {
		log.Println("Kafka URL not configured, skipping Kafka consumers setup")
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

	// Set up the HTTP server for subscription API
	setupHTTPServer(cfg, subscriberService, dbService)
}

// setupHTTPServer configures and starts the HTTP server
func setupHTTPServer(cfg config.Config, subscriberService *services.SubscriberService, dbService *services.DatabaseService) {
	router := mux.NewRouter()

	// Apply CORS middleware to all routes
	router.Use(auth.CORSMiddleware(cfg))

	// Create subscription handlers
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriberService, cfg)
	sessionSubscriptionHandler := handlers.NewSessionSubscriptionHandler(subscriberService, cfg)

	// Event subscription API routes with authentication
	eventApiRouter := router.PathPrefix("/api/scheduler/subscription/v1").Subrouter()
	eventApiRouter.Use(auth.AuthMiddleware)

	// Regular user endpoints for event subscriptions
	eventApiRouter.HandleFunc("/subscribe", subscriptionHandler.Subscribe).Methods("POST")
	eventApiRouter.HandleFunc("/unsubscribe/{eventId}", subscriptionHandler.Unsubscribe).Methods("DELETE")
	eventApiRouter.HandleFunc("/is-subscribed/{eventId}", subscriptionHandler.IsSubscribed).Methods("GET")
	eventApiRouter.HandleFunc("/user-subscriptions", subscriptionHandler.GetUserSubscriptions).Methods("GET")

	// Admin endpoints for event subscriptions with additional middleware
	eventAdminRouter := eventApiRouter.PathPrefix("/event-subscribers").Subrouter()
	eventAdminRouter.Use(auth.AdminMiddleware)
	eventAdminRouter.HandleFunc("/{eventId}", subscriptionHandler.GetEventSubscribers).Methods("GET")

	// Session subscription API routes with authentication
	sessionApiRouter := router.PathPrefix("/api/scheduler/session-subscription/v1").Subrouter()
	sessionApiRouter.Use(auth.AuthMiddleware)

	// Regular user endpoints for session subscriptions
	sessionApiRouter.HandleFunc("/subscribe", sessionSubscriptionHandler.Subscribe).Methods("POST")
	sessionApiRouter.HandleFunc("/unsubscribe/{sessionId}", sessionSubscriptionHandler.Unsubscribe).Methods("DELETE")
	sessionApiRouter.HandleFunc("/is-subscribed/{sessionId}", sessionSubscriptionHandler.IsSubscribed).Methods("GET")
	sessionApiRouter.HandleFunc("/user-subscriptions", sessionSubscriptionHandler.GetUserSubscriptions).Methods("GET")

	// Admin endpoints for session subscriptions with additional middleware
	sessionAdminRouter := sessionApiRouter.PathPrefix("/session-subscribers").Subrouter()
	sessionAdminRouter.Use(auth.AdminMiddleware)
	sessionAdminRouter.HandleFunc("/{sessionId}", sessionSubscriptionHandler.GetSessionSubscribers).Methods("GET")

	// Create health handler for health check endpoints
	healthHandler := handlers.NewHealthHandler(dbService)

	// Healthcheck endpoints (no authentication required)
	router.HandleFunc("/api/scheduler/health", healthHandler.HandleHealth).Methods("GET")

	// K8s probe endpoints
	router.HandleFunc("/healthz", healthHandler.HandleHealth).Methods("GET")   // General health endpoint for both liveness and readiness
	router.HandleFunc("/readyz", healthHandler.HandleReadiness).Methods("GET") // Specific readiness probe endpoint
	router.HandleFunc("/livez", healthHandler.HandleLiveness).Methods("GET")   // Specific liveness probe endpoint	// Start HTTP server
	serverAddr := cfg.ServerHost + ":" + cfg.ServerPort
	log.Printf("Starting HTTP server on %s", serverAddr)

	server := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	log.Fatal(server.ListenAndServe())
}

// testGetUserEmail tests the GetUserEmailByID function with the provided user ID
func testGetUserEmail(cfg config.Config, httpClient *http.Client, userID string) {
	log.Printf("Testing GetUserEmailByID with user ID: %s", userID)

	email, err := auth.GetUserEmailByID(cfg, httpClient, userID)
	if err != nil {
		log.Printf("Error getting email for user %s: %v", userID, err)
		return
	}

	log.Printf("Successfully retrieved email for user %s: %s", userID, email)
}
