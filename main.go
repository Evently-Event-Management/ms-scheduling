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
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

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

		// Check the consolidated scheduling queue for messages (batch)
		rawMessages, err := sqsutil.ReceiveMessage(sqsClient, cfg.SQSSessionSchedulingQueueURL)
		if err != nil {
			log.Printf("Error receiving messages from scheduling SQS queue: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if len(rawMessages) == 0 {
			log.Println("No messages received from scheduling queue, continuing loop.")
			continue // No need to sleep, long polling already waited
		}

		log.Printf("Received %d messages from scheduling queue.", len(rawMessages))
		var messagesToDelete []types.DeleteMessageBatchRequestEntry
		var token string
		var tokenErr error

		// Process each message in the batch
		for _, rawMessage := range rawMessages {
			// Unmarshal and process each message individually
			var messageBody models.SQSMessageBody
			if err := json.Unmarshal([]byte(*rawMessage.Body), &messageBody); err != nil {
				log.Printf("Error unmarshalling message body, will delete malformed message: %v", err)
				// Add malformed message to the delete batch
				messagesToDelete = append(messagesToDelete, types.DeleteMessageBatchRequestEntry{
					Id:            rawMessage.MessageId,
					ReceiptHandle: rawMessage.ReceiptHandle,
				})
				continue
			}

			log.Printf("Processing SQS message from scheduling queue: %+v", messageBody)

			// Get token only once for the batch, if needed
			if token == "" {
				token, tokenErr = auth.GetM2MToken(cfg, httpClient)
				if tokenErr != nil {
					log.Printf("Error getting M2M token: %v. Will retry later.", tokenErr)
					break // Skip processing the rest of the messages if we can't get a token
				}
			}

			// Process the message based on its action
			err = session.ProcessSessionMessage(cfg, httpClient, token, &messageBody)
			if err != nil {
				log.Printf("Error processing %s message for session %s, it will be retried: %v",
					messageBody.Action, messageBody.SessionID, err)
				// If processing fails, DO NOT add it to the delete batch.
				// It will become visible again on the queue for another attempt.
			} else {
				log.Printf("Successfully processed %s message, adding to delete batch.", messageBody.Action)
				// On success, add the message to our list of messages to delete.
				messagesToDelete = append(messagesToDelete, types.DeleteMessageBatchRequestEntry{
					Id:            rawMessage.MessageId,
					ReceiptHandle: rawMessage.ReceiptHandle,
				})
			}
		}

		// After processing the whole batch, delete the successful ones in a single API call
		if len(messagesToDelete) > 0 {
			err := sqsutil.DeleteMessageBatch(cfg.SQSSessionSchedulingQueueURL, sqsClient, messagesToDelete)
			if err != nil {
				log.Printf("Error batch deleting messages: %v", err)
			}
		}
	}
}
