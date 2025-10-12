package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"ms-scheduling/internal/auth"
	"ms-scheduling/internal/config"
	"ms-scheduling/internal/models"
	"ms-scheduling/internal/sqsutil"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// SessionProcessor handles processing of session scheduling messages from SQS
type Processor struct {
	sqsClient       *sqs.Client
	httpClient      *http.Client
	cfg             config.Config
	queueURL        string
	eventServiceURL string
}

// NewProcessor creates a new session scheduling processor
func NewProcessor(sqsClient *sqs.Client, httpClient *http.Client, cfg config.Config) *Processor {
	return &Processor{
		sqsClient:       sqsClient,
		httpClient:      httpClient,
		cfg:             cfg,
		queueURL:        cfg.SQSSessionSchedulingQueueURL,
		eventServiceURL: cfg.EventServiceURL,
	}
}

// ProcessMessages processes messages from the session scheduling queue
func (p *Processor) ProcessMessages(ctx context.Context) error {
	if p.queueURL == "" {
		log.Println("Session scheduling queue URL not configured, skipping processor")
		return fmt.Errorf("session scheduling queue URL not configured")
	}

	log.Printf("Starting to process session scheduling messages from %s", p.queueURL)

	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping session scheduling processor")
			return ctx.Err()
		default:
			// Continue processing
		}

		// Receive messages from scheduling queue
		rawMessages, err := sqsutil.ReceiveMessage(p.sqsClient, p.queueURL)
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
				token, tokenErr = auth.GetM2MToken(p.cfg, p.httpClient)
				if tokenErr != nil {
					log.Printf("Error getting M2M token: %v. Will retry later.", tokenErr)
					break // Skip processing the rest of the messages if we can't get a token
				}
			}

			// Process the message based on its action
			err = p.processSessionMessage(token, &messageBody)
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
			err := sqsutil.DeleteMessageBatch(p.queueURL, p.sqsClient, messagesToDelete)
			if err != nil {
				log.Printf("Error batch deleting messages: %v", err)
			}
		}
	}
}

// processSessionMessage makes the API call to the Event Service to update the session status
func (p *Processor) processSessionMessage(token string, msg *models.SQSMessageBody) error {
	var apiPath string

	switch msg.Action {
	case "ON_SALE":
		apiPath = fmt.Sprintf("/internal/v1/sessions/%s/on-sale", msg.SessionID)
	case "CLOSED":
		apiPath = fmt.Sprintf("/internal/v1/sessions/%s/closed", msg.SessionID)
	default:
		return fmt.Errorf("unknown action in session scheduling message: %s", msg.Action)
	}

	apiURL := p.eventServiceURL + apiPath
	log.Printf("Calling Event Service API: %s", apiURL)

	req, _ := http.NewRequest("PATCH", apiURL, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Printf("HTTP request to Event Service failed: %v", err)
		return err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("Error closing response body: %v", cerr)
		}
	}()

	log.Printf("Event Service response status: %s", resp.Status)
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Event Service response body: %s", string(bodyBytes))

		// Special handling for 404 errors - if the session is not found, we consider the message processed
		// This prevents an infinite loop of retrying non-existent sessions
		if resp.StatusCode == http.StatusNotFound {
			log.Printf("Session %s not found (404). Treating as successfully processed to avoid infinite retries.", msg.SessionID)
			return nil
		}

		if resp.StatusCode == http.StatusConflict {
			log.Printf("Session %s is in a conflicting state (409). Treating as successfully processed to avoid infinite retries.", msg.SessionID)
			return nil
		}

		return fmt.Errorf("API call failed with status %s: %s", resp.Status, string(bodyBytes))
	}

	log.Printf("Successfully processed action '%s' for session %s", msg.Action, msg.SessionID)
	return nil
}
