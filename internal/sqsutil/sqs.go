package sqsutil

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

func ReceiveMessage(sqsClient *sqs.Client, queueURL string) ([]types.Message, error) {
	result, err := sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
		QueueUrl:            &queueURL,
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     20,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to receive message, %v", err)
	}

	// The result is already a slice, so we can return it directly
	return result.Messages, nil
}

func DeleteMessage(queueURL string, client *sqs.Client, receiptHandle *string) {
	log.Printf("Deleting message from SQS queue %s (receipt handle: %s)", queueURL, *receiptHandle)
	_, err := client.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
		QueueUrl:      &queueURL,
		ReceiptHandle: receiptHandle,
	})
	if err != nil {
		log.Printf("Error deleting message from SQS queue %s: %v", queueURL, err)
	} else {
		log.Printf("Message deleted from SQS queue %s successfully", queueURL)
	}
}

func DeleteMessageBatch(queueURL string, client *sqs.Client, entries []types.DeleteMessageBatchRequestEntry) error {
	if len(entries) == 0 {
		return nil
	}

	log.Printf("Deleting %d messages in a batch from SQS queue %s", len(entries), queueURL)
	result, err := client.DeleteMessageBatch(context.TODO(), &sqs.DeleteMessageBatchInput{
		QueueUrl: aws.String(queueURL),
		Entries:  entries,
	})

	if err != nil {
		return fmt.Errorf("batch delete failed: %v", err)
	}

	if len(result.Failed) > 0 {
		log.Printf("Warning: %d messages failed to delete in batch operation", len(result.Failed))
		for _, failure := range result.Failed {
			log.Printf("Delete failure - ID: %s, Code: %s, Message: %s",
				*failure.Id, *failure.Code, *failure.Message)
		}
	}

	log.Printf("Successfully deleted %d messages in batch", len(result.Successful))
	return nil
}
