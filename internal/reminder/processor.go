package reminder

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ms-scheduling/internal/config"
	"ms-scheduling/internal/models"
	"ms-scheduling/internal/services"
	"ms-scheduling/internal/sqsutil"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// Processor handles processing of reminder messages from SQS
type Processor struct {
	sqsClient         *sqs.Client
	httpClient        *http.Client
	cfg               config.Config
	queueURL          string
	subscriberService *services.SubscriberService
}

// NewProcessor creates a new reminder processor
func NewProcessor(sqsClient *sqs.Client, httpClient *http.Client, cfg config.Config, subscriberService *services.SubscriberService) *Processor {
	return &Processor{
		sqsClient:         sqsClient,
		httpClient:        httpClient,
		cfg:               cfg,
		queueURL:          cfg.SQSSessionRemindersQueueURL,
		subscriberService: subscriberService,
	}
}

// ProcessMessages processes messages from the reminder queue
func (p *Processor) ProcessMessages(ctx context.Context) error {
	if p.queueURL == "" {
		log.Println("Reminder queue URL not configured, skipping reminder processor")
		return fmt.Errorf("reminder queue URL not configured")
	}

	log.Printf("Starting to process reminder messages from %s", p.queueURL)

	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping reminder processor")
			return ctx.Err()
		default:
			// Continue processing
		}

		// Receive messages from reminder queue
		rawMessages, err := sqsutil.ReceiveMessage(p.sqsClient, p.queueURL)
		if err != nil {
			log.Printf("Error receiving messages from reminder SQS queue: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if len(rawMessages) == 0 {
			log.Println("No messages received from reminder queue, continuing loop.")
			continue // No need to sleep, long polling already waited
		}

		log.Printf("Received %d messages from reminder queue.", len(rawMessages))
		var messagesToDelete []types.DeleteMessageBatchRequestEntry

		// Process each message in the batch
		for _, rawMessage := range rawMessages {
			// Unmarshal and process each message individually
			var messageBody models.SQSReminderMessageBody
			if err := json.Unmarshal([]byte(*rawMessage.Body), &messageBody); err != nil {
				log.Printf("Error unmarshalling reminder message body, will delete malformed message: %v", err)
				// Add malformed message to the delete batch
				messagesToDelete = append(messagesToDelete, types.DeleteMessageBatchRequestEntry{
					Id:            rawMessage.MessageId,
					ReceiptHandle: rawMessage.ReceiptHandle,
				})
				continue
			}

			log.Printf("Processing SQS message from reminder queue: %+v", messageBody)

			// Process the reminder message
			err = p.processReminderMessage(&messageBody)
			if err != nil {
				log.Printf("Error processing reminder for session %s, it will be retried: %v",
					messageBody.SessionID, err)
				// If processing fails, DO NOT add it to the delete batch.
				// It will become visible again on the queue for another attempt.
			} else {
				log.Printf("Successfully processed reminder message for session %s, adding to delete batch.", messageBody.SessionID)
				// On success, add the message to our list of messages to delete.
				messagesToDelete = append(messagesToDelete, types.DeleteMessageBatchRequestEntry{
					Id:            rawMessage.MessageId,
					ReceiptHandle: rawMessage.ReceiptHandle,
				})
			}
		}

		// After processing the whole batch, delete the successful ones in a single API call
		if len(messagesToDelete) > 0 {
			err := sqsutil.DeleteMessageBatch(p.queueURL, p.sqsClient, messagesToDelete)
			if err != nil {
				log.Printf("Error batch deleting reminder messages: %v", err)
			}
		}
	}
}

// processReminderMessage handles sending emails for session reminders
func (p *Processor) processReminderMessage(msg *models.SQSReminderMessageBody) error {
	if msg.Action != "REMINDER_EMAIL" {
		return fmt.Errorf("unknown action in reminder message: %s", msg.Action)
	}

	log.Printf("Processing reminder email for session %s (type: %s)", msg.SessionID, msg.ReminderType)

	// Process the session reminder through the subscriber service
	err := p.subscriberService.ProcessSessionReminder(msg.SessionID)
	if err != nil {
		log.Printf("Error sending reminder emails for session %s: %v", msg.SessionID, err)
		return err
	}

	log.Printf("Successfully sent reminder emails for session %s", msg.SessionID)
	return nil
}
