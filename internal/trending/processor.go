package trending

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"ms-scheduling/internal/auth"
	"ms-scheduling/internal/config"
	"ms-scheduling/internal/sqsutil"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// TrendingProcessor handles processing of trending calculation jobs from SQS
type Processor struct {
	sqsClient         *sqs.Client
	httpClient        *http.Client
	cfg               config.Config
	queueURL          string
	eventQueryBaseURL string
}

// Message represents a trending job message from SQS
type Message struct {
	// Add any specific fields that might be in the trending job messages
	Action    string `json:"action"`
	Timestamp string `json:"timestamp"`
}

// NewProcessor creates a new trending job processor
func NewProcessor(sqsClient *sqs.Client, httpClient *http.Client, cfg config.Config) *Processor {
	return &Processor{
		sqsClient:         sqsClient,
		httpClient:        httpClient,
		cfg:               cfg,
		queueURL:          cfg.SQSTrendingQueueURL,
		eventQueryBaseURL: cfg.EventQueryServiceURL,
	}
}

// ProcessMessages processes messages from the trending queue
func (p *Processor) ProcessMessages(ctx context.Context) error {
	if p.queueURL == "" {
		log.Println("Trending queue URL not configured, skipping trending job processing")
		return fmt.Errorf("trending queue URL not configured")
	}

	log.Printf("Starting to process trending job messages from %s", p.queueURL)

	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping trending job processing")
			return ctx.Err()
		default:
			// Continue processing
		}

		// Receive messages from trending queue
		rawMessages, err := sqsutil.ReceiveMessage(p.sqsClient, p.queueURL)
		if err != nil {
			log.Printf("Error receiving messages from trending SQS queue: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if len(rawMessages) == 0 {
			log.Println("No messages received from trending queue, continuing loop.")
			continue // No need to sleep, long polling already waited
		}

		log.Printf("Received %d messages from trending queue.", len(rawMessages))
		var messagesToDelete []types.DeleteMessageBatchRequestEntry
		var token string
		var tokenErr error

		// Process each message in the batch
		for _, rawMessage := range rawMessages {
			log.Printf("Processing trending job message: %s", *rawMessage.Body)

			// Get token only once for the batch, if needed
			if token == "" {
				token, tokenErr = auth.GetM2MToken(p.cfg, p.httpClient)
				if tokenErr != nil {
					log.Printf("Error getting M2M token for trending job: %v. Will retry later.", tokenErr)
					break // Skip processing the rest of the messages if we can't get a token
				}
			}

			// Process the message
			err = p.processTrendingMessage(token, *rawMessage.Body)
			if err != nil {
				log.Printf("Error processing trending job message: %v, it will be retried", err)
				// If processing fails, DO NOT add it to the delete batch.
				// It will become visible again on the queue for another attempt.
			} else {
				log.Printf("Successfully processed trending job message, adding to delete batch.")
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
				log.Printf("Error batch deleting trending job messages: %v", err)
			}
		}
	}
}

// processTrendingMessage processes a single trending job message
func (p *Processor) processTrendingMessage(token, messageBody string) error {
	// Parse the message body if needed - adjust based on your actual message structure
	var message Message
	err := json.Unmarshal([]byte(messageBody), &message)
	if err != nil {
		return fmt.Errorf("error unmarshalling trending message: %w", err)
	}

	// Call the event query service to calculate trends
	return p.calculateTrends(token)
}

// calculateTrends calls the event query service to calculate trending events
func (p *Processor) calculateTrends(token string) error {
	endpoint := fmt.Sprintf("%s/internal/v1/trending/calculate-all", p.eventQueryBaseURL)
	log.Printf("Sending request to calculate trends: %s", endpoint)

	// Create an empty request body or customize as needed
	reqBody := []byte("{}")

	// Create the request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Send the request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("Error closing response body: %v", cerr)
		}
	}()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("trending calculation request failed with status %d: %s",
			resp.StatusCode, string(bodyBytes))
	}

	log.Printf("Trending calculation request successful with status: %s", resp.Status)
	return nil
}
