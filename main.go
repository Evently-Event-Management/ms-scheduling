package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awsscheduler "github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	auth "ms-scheduling/internal/auth"
	appconfig "ms-scheduling/internal/config"
	"ms-scheduling/internal/kafka"
	"ms-scheduling/internal/models"
	"ms-scheduling/internal/scheduler"
	"ms-scheduling/internal/session"
	"ms-scheduling/internal/sqsutil"
)

// Types moved to internal packages.

// Main application loop
func main() {
	cfg := appconfig.Load()
	log.Printf("Loaded config: %+v", cfg)

	// Create clients once, outside the loop
	httpClient := &http.Client{Timeout: 10 * time.Second}
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
	schedulerService := scheduler.NewService(cfg, schedulerClient)

	// Start Kafka consumer in a separate goroutine if Kafka URL is configured
	if cfg.KafkaURL != "" && cfg.KafkaTopic != "" {
		log.Printf("Starting Kafka consumer for topic %s at %s", cfg.KafkaTopic, cfg.KafkaURL)
		kafkaConsumer := kafka.NewConsumer(cfg, cfg.KafkaURL, cfg.KafkaTopic, schedulerService)
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

	for {
		log.Println("Starting main loop iteration")

		// We only need the token if we have a message to process.
		// Use separate variables for ON_SALE and CLOSED to avoid accidental reuse.

		// Check the consolidated scheduling queue
		rawMessage, err := sqsutil.ReceiveMessage(sqsClient, cfg.SQSSessionSchedulingQueueURL)
		if err != nil {
			log.Printf("Error receiving message from scheduling SQS queue: %v", err)
			time.Sleep(5 * time.Second) // Wait before retrying
			continue
		}

		if rawMessage != nil {
			// Unmarshal the body here
			var messageBody models.SQSMessageBody
			if err := json.Unmarshal([]byte(*rawMessage.Body), &messageBody); err != nil {
				log.Printf("Error unmarshalling SQS message body, deleting malformed message: %v", err)
				// This is a "poison pill" message, delete it so it doesn't block the queue
				sqsutil.DeleteMessage(cfg.SQSSessionSchedulingQueueURL, sqsClient, rawMessage.ReceiptHandle)
				continue
			}

			log.Printf("Received SQS message from scheduling queue: %+v", messageBody)

			// Get token only when we need it
			token, err := auth.GetM2MToken(cfg, httpClient)
			if err != nil {
				log.Printf("Error getting M2M token: %v. Retrying in 30 seconds.", err)
				time.Sleep(30 * time.Second)
				continue
			}

			// Process the message based on its action
			err = session.ProcessSessionMessage(cfg, httpClient, token, &messageBody)
			if err != nil {
				log.Printf("Error processing %s message for session %s, will retry: %v",
					messageBody.Action, messageBody.SessionID, err)
			} else {
				// SUCCESS! Now we can safely delete the message.
				log.Printf("Successfully processed %s message, deleting from queue.", messageBody.Action)
				sqsutil.DeleteMessage(cfg.SQSSessionSchedulingQueueURL, sqsClient, rawMessage.ReceiptHandle)
			}
			continue // Process one message at a time
		}

		log.Println("No messages received from scheduling queue, sleeping for a moment.")
		time.Sleep(1 * time.Second) // Small sleep to prevent a tight loop when idle
	}
}
