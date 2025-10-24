package reminder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

var errResourceNotFound = errors.New("resource not found")

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
	// Validate message basics
	if msg.SessionID == "" {
		log.Printf("Reminder message has empty SessionID, skipping: %+v", msg)
		return nil // Return nil to delete the message from queue
	}

	log.Printf("Processing reminder email for session %s (type: %s, template: %s, notification ID: %s)",
		msg.SessionID, msg.ReminderType, msg.TemplateID, msg.NotificationID)

	// Handle based solely on ReminderType
	switch msg.ReminderType {
	case "SESSION_START":
		return p.handleReminder(msg.SessionID, func(subscribers []models.Subscriber, info *services.SessionReminderInfo) error {
			return p.subscriberService.SendSessionStartReminderEmails(subscribers, info)
		})

	case "SALE_START":
		return p.handleReminder(msg.SessionID, func(subscribers []models.Subscriber, info *services.SessionReminderInfo) error {
			return p.subscriberService.SendSessionSalesReminderEmails(subscribers, info)
		})
	default:
		// For unknown reminder types, log and delete from queue (return nil)
		log.Printf("Unknown reminder type: %s, skipping. Full message: %+v", msg.ReminderType, msg)
		return nil
	}
}

func (p *Processor) handleReminder(sessionID string, send func([]models.Subscriber, *services.SessionReminderInfo) error) error {
	subscribers, sessionInfo, err := p.prepareSessionReminderData(sessionID)
	if err != nil {
		if errors.Is(err, errResourceNotFound) {
			log.Printf("Session %s not found. Consuming reminder message without sending emails.", sessionID)
			return nil
		}
		return err
	}

	if len(subscribers) == 0 {
		log.Printf("No subscribers found for session %s reminder", sessionID)
		return nil
	}

	if err := send(subscribers, sessionInfo); err != nil {
		return fmt.Errorf("failed to send reminder emails for session %s: %w", sessionID, err)
	}

	return nil
}

func (p *Processor) prepareSessionReminderData(sessionID string) ([]models.Subscriber, *services.SessionReminderInfo, error) {
	sessionDetails, err := p.fetchSessionExtendedInfo(sessionID)
	if err != nil {
		return nil, nil, err
	}

	sessionInfo := &services.SessionReminderInfo{
		SessionID:      sessionDetails.SessionID,
		EventID:        sessionDetails.EventID,
		EventTitle:     sessionDetails.EventTitle,
		StartTime:      sessionDetails.StartTime.UnixMicro(),
		EndTime:        sessionDetails.EndTime.UnixMicro(),
		Status:         sessionDetails.Status,
		VenueDetails:   "",
		SessionType:    sessionDetails.SessionType,
		SalesStartTime: sessionDetails.SalesStartTime.UnixMicro(),
	}

	if venueBytes, err := json.Marshal(sessionDetails.VenueDetails); err == nil {
		sessionInfo.VenueDetails = string(venueBytes)
	} else {
		log.Printf("Warning: Failed to marshal venue details for session %s: %v", sessionID, err)
	}

	// Fetch event details from event-query service
	if sessionInfo.EventID != "" {
		eventDetails, err := p.fetchEventBasicInfo(sessionInfo.EventID)
		if err == nil {
			sessionInfo.EventTitle = eventDetails.Title
			sessionInfo.EventDescription = eventDetails.Description
			sessionInfo.EventOverview = eventDetails.Overview
			sessionInfo.EventCoverPhotos = eventDetails.CoverPhotos
			if eventDetails.Organization.Name != "" {
				sessionInfo.OrganizationName = eventDetails.Organization.Name
				sessionInfo.OrganizationLogo = eventDetails.Organization.LogoURL
			}
			if eventDetails.Category.Name != "" {
				sessionInfo.CategoryName = eventDetails.Category.Name
			}
		} else if errors.Is(err, errResourceNotFound) {
			log.Printf("Event %s not found while preparing reminder for session %s", sessionInfo.EventID, sessionID)
		} else {
			log.Printf("Warning: Could not fetch event details for event %s: %v", sessionInfo.EventID, err)
		}
	}

	sessionSubscribers, err := p.subscriberService.GetSessionSubscribers(sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting session subscribers: %w", err)
	}

	var eventSubscribers []models.Subscriber
	if sessionInfo.EventID != "" {
		eventSubscribers, err = p.subscriberService.GetEventSubscribers(sessionInfo.EventID)
		if err != nil {
			log.Printf("Warning: Could not get event subscribers for event %s: %v", sessionInfo.EventID, err)
		}
	}

	allSubscribers := combineAndDeduplicateSubscribers(sessionSubscribers, eventSubscribers)

	return allSubscribers, sessionInfo, nil
}

func (p *Processor) fetchSessionExtendedInfo(sessionID string) (*models.SessionExtendedInfo, error) {
	if p.cfg.EventQueryServiceURL == "" {
		return nil, fmt.Errorf("event query service URL not configured")
	}

	apiURL := fmt.Sprintf("%s/v1/events/sessions/%s/extended-info", p.cfg.EventQueryServiceURL, sessionID)
	log.Printf("Fetching session details from: %s", apiURL)

	resp, err := p.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch session info: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("Error closing session info response body: %v", cerr)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errResourceNotFound
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("session info API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var sessionInfo models.SessionExtendedInfo
	if err := json.NewDecoder(resp.Body).Decode(&sessionInfo); err != nil {
		return nil, fmt.Errorf("failed to decode session info: %w", err)
	}

	return &sessionInfo, nil
}

func (p *Processor) fetchEventBasicInfo(eventID string) (*models.EventBasicInfo, error) {
	if p.cfg.EventQueryServiceURL == "" {
		return nil, fmt.Errorf("event query service URL not configured")
	}

	apiURL := fmt.Sprintf("%s/v1/events/%s/basic-info", p.cfg.EventQueryServiceURL, eventID)
	log.Printf("Fetching event details from: %s", apiURL)

	resp, err := p.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch event info: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("Error closing event info response body: %v", cerr)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errResourceNotFound
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("event info API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var eventInfo models.EventBasicInfo
	if err := json.NewDecoder(resp.Body).Decode(&eventInfo); err != nil {
		return nil, fmt.Errorf("failed to decode event info: %w", err)
	}

	return &eventInfo, nil
}

func combineAndDeduplicateSubscribers(sessionSubs, eventSubs []models.Subscriber) []models.Subscriber {
	if len(sessionSubs) == 0 && len(eventSubs) == 0 {
		return nil
	}

	subscriberMap := make(map[int]models.Subscriber, len(sessionSubs)+len(eventSubs))

	for _, sub := range sessionSubs {
		subscriberMap[sub.SubscriberID] = sub
	}

	for _, sub := range eventSubs {
		subscriberMap[sub.SubscriberID] = sub
	}

	result := make([]models.Subscriber, 0, len(subscriberMap))
	for _, sub := range subscriberMap {
		result = append(result, sub)
	}

	return result
}
