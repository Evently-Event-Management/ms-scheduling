package sqsutil

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

func ReceiveMessage(sqsClient *sqs.Client, queueURL string) (*types.Message, error) {
	result, err := sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
		QueueUrl:            &queueURL,
		MaxNumberOfMessages: 1,
		WaitTimeSeconds:     20,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to receive message, %v", err)
	}

	if len(result.Messages) == 0 {
		return nil, nil // No message received, this is normal
	}

	return &result.Messages[0], nil
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
