package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func main() {
	fmt.Println("🗑️  SQS Queue Drain Utility")
	fmt.Println("===========================")

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("ap-south-1"),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Create SQS client
	sqsClient := sqs.NewFromConfig(cfg)

	// SQS queue URL from your configuration
	queueURL := "https://sqs.ap-south-1.amazonaws.com/621014405736/session-scheduling-queue-infra-dev-isurumuni"

	fmt.Printf("🎯 Target Queue: %s\n", queueURL)
	fmt.Println("🔄 Draining messages by receiving and deleting them...")

	deletedCount := 0
	maxMessages := int32(10) // Process up to 10 messages per batch

	for {
		// Receive messages
		result, err := sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
			QueueUrl:            &queueURL,
			MaxNumberOfMessages: maxMessages,
			WaitTimeSeconds:     2, // Short polling
			VisibilityTimeout:   30,
		})

		if err != nil {
			log.Printf("❌ Error receiving messages: %v", err)
			break
		}

		// If no messages, we're done
		if len(result.Messages) == 0 {
			fmt.Println("✅ No more messages in queue")
			break
		}

		// Delete each message
		for _, msg := range result.Messages {
			// Print message details
			fmt.Printf("📤 Deleting message: %s\n", *msg.Body)
			
			_, err := sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
				QueueUrl:      &queueURL,
				ReceiptHandle: msg.ReceiptHandle,
			})

			if err != nil {
				log.Printf("❌ Error deleting message: %v", err)
			} else {
				deletedCount++
			}
		}

		fmt.Printf("🗑️  Deleted %d messages in this batch\n", len(result.Messages))
		
		// Small delay between batches
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("✅ Queue drain completed! Deleted %d messages total.\n", deletedCount)
	fmt.Println("🔄 You can now send new messages with UUID format.")
}