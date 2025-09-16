package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	auth "ms-scheduling/internal/auth"
	appconfig "ms-scheduling/internal/config"
	"ms-scheduling/internal/models"
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

	for {
		log.Println("Starting main loop iteration")

		// We only need the token if we have a message to process.
		// Use separate variables for ON_SALE and CLOSED to avoid accidental reuse.

		// Check ON_SALE queue
		rawOnSaleMessage, err := sqsutil.ReceiveMessage(sqsClient, cfg.SQSONSaleQueueURL)
		if err != nil {
			log.Printf("Error receiving message from ON_SALE SQS queue: %v", err)
			time.Sleep(5 * time.Second) // Wait before retrying
			continue
		}

		if rawOnSaleMessage != nil {
			// Unmarshal the body here
			var onSaleBody models.SQSMessageBody
			if err := json.Unmarshal([]byte(*rawOnSaleMessage.Body), &onSaleBody); err != nil {
				log.Printf("Error unmarshalling ON_SALE message body, deleting malformed message: %v", err)
				// This is a "poison pill" message, delete it so it doesn't block the queue
				sqsutil.DeleteMessage(cfg.SQSONSaleQueueURL, sqsClient, rawOnSaleMessage.ReceiptHandle)
				continue
			}

			log.Printf("Received SQS message from ON_SALE queue: %+v", onSaleBody)

			// Get token only when we need it
			onSaleToken, err := auth.GetM2MToken(cfg, httpClient)
			if err != nil {
				log.Printf("Error getting M2M token: %v. Retrying in 30 seconds.", err)
				time.Sleep(30 * time.Second)
				continue
			}

			// Process the message
			err = session.ProcessSessionMessage(cfg, httpClient, onSaleToken, &onSaleBody)
			if err != nil {
				log.Printf("Error processing ON_SALE message for session %s, will retry: %v", onSaleBody.SessionID, err)
			} else {
				// SUCCESS! Now we can safely delete the message.
				log.Printf("Successfully processed ON_SALE message, deleting from queue.")
				sqsutil.DeleteMessage(cfg.SQSONSaleQueueURL, sqsClient, rawOnSaleMessage.ReceiptHandle)
			}
			continue // Process one message at a time
		}

		// Check CLOSED queue (apply the same logic)
		rawClosedMessage, err := sqsutil.ReceiveMessage(sqsClient, cfg.SQSSClosedQueueURL)
		if err != nil {
			log.Printf("Error receiving message from CLOSED SQS queue: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if rawClosedMessage != nil {
			var closedBody models.SQSMessageBody
			if err := json.Unmarshal([]byte(*rawClosedMessage.Body), &closedBody); err != nil {
				log.Printf("Error unmarshalling CLOSED message body, deleting malformed message: %v", err)
				sqsutil.DeleteMessage(cfg.SQSSClosedQueueURL, sqsClient, rawClosedMessage.ReceiptHandle)
				continue
			}

			log.Printf("Received SQS message from CLOSED queue: %+v", closedBody)

			closedToken, err := auth.GetM2MToken(cfg, httpClient)
			if err != nil {
				log.Printf("Error getting M2M token: %v. Retrying in 30 seconds.", err)
				time.Sleep(30 * time.Second)
				continue
			}

			err = session.ProcessSessionMessage(cfg, httpClient, closedToken, &closedBody)
			if err != nil {
				log.Printf("Error processing CLOSED message for session %s, will retry: %v", closedBody.SessionID, err)
			} else {
				log.Printf("Successfully processed CLOSED message, deleting from queue.")
				sqsutil.DeleteMessage(cfg.SQSSClosedQueueURL, sqsClient, rawClosedMessage.ReceiptHandle)
			}
			continue
		}

		log.Println("No messages received from either queue, sleeping for a moment.")
		time.Sleep(1 * time.Second) // Small sleep to prevent a tight loop when idle
	}
}
