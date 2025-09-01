package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	_ "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// Config holds all the configuration for our service, loaded from environment variables.
type Config struct {
	AWSRegion          string
	AWSEndpoint        string // For LocalStack
	EventServiceURL    string
	KeycloakURL        string
	KeycloakRealm      string
	ClientID           string
	ClientSecret       string
	SQSONSaleQueueURL  string
	SQSSClosedQueueURL string
}

// M2MTokenResponse defines the structure of the token response from Keycloak.
type M2MTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// SQSMessageBody defines the structure of the message we expect.
type SQSMessageBody struct {
	SessionID string `json:"sessionId"`
	Action    string `json:"action"` // e.g., "ON_SALE", "CLOSED"
}

// Main application loop
func main() {
	cfg := loadConfig()
	log.Printf("Loaded config: %+v", cfg)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	log.Println("HTTP client initialized")

	for {
		log.Println("Starting main loop iteration")

		accessToken, err := getM2MToken(cfg, httpClient)
		if err != nil {
			log.Printf("Error getting M2M token: %v. Retrying in 30 seconds.", err)
			time.Sleep(30 * time.Second)
			continue
		}
		log.Println("Successfully obtained M2M token")

		// Check ON_SALE queue
		onSaleMessage, err := receiveMessage(cfg, cfg.SQSONSaleQueueURL)
		if err != nil {
			log.Printf("Error receiving message from ON_SALE SQS queue: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if onSaleMessage != nil {
			log.Printf("Received SQS message from ON_SALE queue: %+v", onSaleMessage)
			err = processSessionMessage(cfg, httpClient, accessToken, onSaleMessage)
			if err != nil {
				log.Printf("Error processing ON_SALE message for session %s: %v", onSaleMessage.SessionID, err)
			}
			continue // Process one message at a time
		}

		// Check CLOSED queue
		soldOutMessage, err := receiveMessage(cfg, cfg.SQSSClosedQueueURL)
		if err != nil {
			log.Printf("Error receiving message from CLOSED SQS queue: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if soldOutMessage != nil {
			log.Printf("Received SQS message from CLOSED queue: %+v", soldOutMessage)
			err = processSessionMessage(cfg, httpClient, accessToken, soldOutMessage)
			if err != nil {
				log.Printf("Error processing CLOSED message for session %s: %v", soldOutMessage.SessionID, err)
			}
			continue // Process one message at a time
		}

		log.Println("No messages received from either queue, continuing loop")
	}
}

// getM2MToken performs the Client Credentials Grant flow to get a machine-to-machine token.
func getM2MToken(cfg Config, client *http.Client) (string, error) {
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", cfg.KeycloakURL, cfg.KeycloakRealm)
	log.Printf("Requesting M2M token from: %s", tokenURL)

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", cfg.ClientID)
	data.Set("client_secret", cfg.ClientSecret)

	req, _ := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	log.Printf("Sending POST request to Keycloak for token with client_id: %s", cfg.ClientID)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP request to Keycloak failed: %v", err)
		return "", err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("Error closing response body: %v", cerr)
		}
	}()

	log.Printf("Keycloak token response status: %s", resp.Status)
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Keycloak token response body: %s", string(bodyBytes))
		return "", fmt.Errorf("failed to get token, status: %s", resp.Status)
	}

	var tokenResp M2MTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		log.Printf("Error decoding token response: %v", err)
		return "", err
	}
	log.Printf("Received access token: %s", tokenResp.AccessToken)

	return tokenResp.AccessToken, nil
}

// receiveMessage performs a long poll on the specified SQS queue.
func receiveMessage(cfg Config, queueURL string) (*SQSMessageBody, error) {
	awsCfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(cfg.AWSRegion))
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config, %v", err)
	}

	sqsClient := sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		// This block now only runs if you've explicitly set the local endpoint URL.
		if cfg.AWSEndpoint != "" {
			log.Printf("Using LocalStack endpoint for AWS services: %s", cfg.AWSEndpoint)
			o.BaseEndpoint = &cfg.AWSEndpoint
		}
	})

	result, err := sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
		QueueUrl:            &queueURL,
		MaxNumberOfMessages: 1,
		WaitTimeSeconds:     5, // Reduced polling time to check both queues more frequently
	})
	if err != nil {
		return nil, fmt.Errorf("failed to receive message, %v", err)
	}

	if len(result.Messages) == 0 {
		return nil, nil // No message received, this is normal
	}

	message := result.Messages[0]
	var body SQSMessageBody
	if err := json.Unmarshal([]byte(*message.Body), &body); err != nil {
		log.Printf("Error unmarshalling message body: %v", err)
		deleteMessage(queueURL, sqsClient, message.ReceiptHandle)
		return nil, nil
	}

	deleteMessage(queueURL, sqsClient, message.ReceiptHandle)
	return &body, nil
}

// processSessionMessage makes the API call to the Event Service to update the session status.
func processSessionMessage(cfg Config, client *http.Client, token string, msg *SQSMessageBody) error {
	var apiPath string

	// Use a switch to determine the correct API endpoint based on the action.
	switch msg.Action {
	case "ON_SALE":
		apiPath = fmt.Sprintf("/internal/v1/sessions/%s/on-sale", msg.SessionID)
	case "CLOSED":
		apiPath = fmt.Sprintf("/internal/v1/sessions/%s/closed", msg.SessionID) // changed from "sold-out" to "closed"
	default:
		return fmt.Errorf("unknown action in SQS message: %s", msg.Action)
	}

	apiURL := cfg.EventServiceURL + apiPath
	log.Printf("Calling Event Service API: %s", apiURL)

	req, _ := http.NewRequest("PATCH", apiURL, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)

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
		return fmt.Errorf("API call failed with status %s: %s", resp.Status, string(bodyBytes))
	}

	log.Printf("Successfully processed action '%s' for session %s", msg.Action, msg.SessionID)
	return nil
}

// deleteMessage removes a message from the SQS queue.
func deleteMessage(queueURL string, client *sqs.Client, receiptHandle *string) {
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

// loadConfig reads all necessary configuration from environment variables.
func loadConfig() Config {
	log.Println("Loading configuration from environment variables")
	return Config{
		SQSONSaleQueueURL:  getEnv("AWS_SQS_SESSION_ON_SALE_URL", ""),
		SQSSClosedQueueURL: getEnv("AWS_SQS_SESSION_CLOSED_URL", ""),
		AWSRegion:          getEnv("AWS_REGION", "us-east-1"),
		AWSEndpoint:        getEnv("AWS_LOCAL_ENDPOINT_URL", ""),
		EventServiceURL:    getEnv("EVENT_SERVICE_URL", "http://localhost:8081/api/event-seating"),
		KeycloakURL:        getEnv("KEYCLOAK_URL", "https://auth.dpiyumal.me:8080"),
		KeycloakRealm:      getEnv("KEYCLOAK_REALM", "event-ticketing"),
		ClientID:           getEnv("KEYCLOAK_CLIENT_ID", "scheduler-service-client"),
		ClientSecret:       getEnv("SCHEDULER_CLIENT_SECRET", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		log.Printf("Loaded env var %s: %s", key, value)
		return value
	}
	log.Printf("Env var %s not set, using fallback: %s", key, fallback)
	return fallback
}
