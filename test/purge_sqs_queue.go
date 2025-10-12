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
	fmt.Println("🧹 SQS Queue Purge Utility")
	fmt.Println("==========================")

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
	fmt.Println("⚠️  This will delete ALL messages in the queue!")
	fmt.Println("Press Ctrl+C to cancel, or wait 3 seconds to proceed...")

	// Give user time to cancel
	for i := 3; i > 0; i-- {
		fmt.Printf("⏳ Starting in %d seconds...\n", i)
		time.Sleep(1 * time.Second)
	}

	// Purge the queue
	fmt.Println("🧹 Purging SQS queue...")
	_, err = sqsClient.PurgeQueue(context.TODO(), &sqs.PurgeQueueInput{
		QueueUrl: &queueURL,
	})

	if err != nil {
		log.Fatalf("❌ Failed to purge queue: %v", err)
	}

	fmt.Println("✅ SQS queue purged successfully!")
	fmt.Println("🔄 You can now send new messages with UUID format.")
	fmt.Println("💡 Note: It may take up to 60 seconds for the purge to complete.")
}