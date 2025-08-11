package main

import (
	"bytes"
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
	SQSQueueURL     string
	AWSRegion       string
	AWSEndpoint     string // For LocalStack
	EventServiceURL string
	KeycloakURL     string
	KeycloakRealm   string
	ClientID        string
	ClientSecret    string
}

// M2MTokenResponse defines the structure of the token response from Keycloak.
type M2MTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// SQSMessageBody defines the structure of the message we expect from the SQS queue.
type SQSMessageBody struct {
	SessionID string `json:"sessionId"`
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

		message, err := receiveMessage(cfg)
		if err != nil {
			log.Printf("Error receiving message from SQS: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if message == nil {
			log.Println("No message received from SQS, continuing loop")
			continue
		}
		log.Printf("Received SQS message: %+v", message)

		err = processMessage(cfg, httpClient, accessToken, message)
		if err != nil {
			log.Printf("Error processing message for session %s: %v", message.SessionID, err)
		}
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

// receiveMessage performs a long poll on the SQS queue.
func receiveMessage(cfg Config) (*SQSMessageBody, error) {
	log.Printf("Loading AWS config for region: %s", cfg.AWSRegion)
	awsCfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(cfg.AWSRegion))
	if err != nil {
		log.Printf("Unable to load AWS SDK config: %v", err)
		return nil, fmt.Errorf("unable to load AWS SDK config, %v", err)
	}

	log.Printf("Creating SQS client (endpoint: %s)", cfg.AWSEndpoint)
	sqsClient := sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		if cfg.AWSEndpoint != "" {
			o.BaseEndpoint = &cfg.AWSEndpoint
		}
	})

	log.Printf("Polling SQS queue: %s", cfg.SQSQueueURL)
	result, err := sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
		QueueUrl:            &cfg.SQSQueueURL,
		MaxNumberOfMessages: 1,
		WaitTimeSeconds:     20,
	})
	if err != nil {
		log.Printf("Failed to receive message from SQS: %v", err)
		return nil, fmt.Errorf("failed to receive message, %v", err)
	}

	log.Printf("Received %d messages from SQS", len(result.Messages))
	if len(result.Messages) == 0 {
		return nil, nil
	}

	message := result.Messages[0]
	log.Printf("Raw SQS message body: %s", *message.Body)
	var body SQSMessageBody
	if err := json.Unmarshal([]byte(*message.Body), &body); err != nil {
		log.Printf("Error unmarshalling message body: %v", err)
		deleteMessage(cfg, sqsClient, message.ReceiptHandle)
		return nil, nil
	}

	log.Printf("Parsed SQS message body: %+v", body)
	deleteMessage(cfg, sqsClient, message.ReceiptHandle)
	log.Printf("Deleted message from SQS (receipt handle: %s)", *message.ReceiptHandle)

	return &body, nil
}

// processMessage makes the API call to the Event Service to update the session status.
func processMessage(cfg Config, client *http.Client, token string, msg *SQSMessageBody) error {
	apiURL := fmt.Sprintf("%s/internal/v1/sessions/%s/status", cfg.EventServiceURL, msg.SessionID)
	log.Printf("Calling Event Service API: %s", apiURL)

	payload := []byte(`{"status": "ON_SALE"}`)
	log.Printf("PATCH payload: %s", string(payload))
	req, _ := http.NewRequest("PATCH", apiURL, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	log.Printf("Sending PATCH request to Event Service for session %s", msg.SessionID)
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

	log.Printf("Successfully processed session %s, status set to ON_SALE", msg.SessionID)
	return nil
}

// deleteMessage removes a message from the SQS queue.
func deleteMessage(cfg Config, client *sqs.Client, receiptHandle *string) {
	log.Printf("Deleting message from SQS (receipt handle: %s)", *receiptHandle)
	_, err := client.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
		QueueUrl:      &cfg.SQSQueueURL,
		ReceiptHandle: receiptHandle,
	})
	if err != nil {
		log.Printf("Error deleting message from SQS: %v", err)
	} else {
		log.Printf("Message deleted from SQS successfully")
	}
}

// loadConfig reads all necessary configuration from environment variables.
func loadConfig() Config {
	log.Println("Loading configuration from environment variables")
	return Config{
		// ✅ FIXED: Use the correct LocalStack SQS URL format
		SQSQueueURL:     getEnv("SQS_QUEUE_URL", "http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/session-on-sale-queue"),
		AWSRegion:       getEnv("AWS_REGION", "us-east-1"),
		AWSEndpoint:     getEnv("AWS_ENDPOINT_URL", "http://localhost:4566"),
		EventServiceURL: getEnv("EVENT_SERVICE_URL", "http://localhost:8081/api/event-seating"),
		KeycloakURL:     getEnv("KEYCLOAK_URL", "http://localhost:8080"),
		KeycloakRealm:   getEnv("KEYCLOAK_REALM", "event-ticketing"),
		ClientID:        getEnv("KEYCLOAK_CLIENT_ID", "scheduler-service-client"),
		ClientSecret:    getEnv("SCHEDULER_CLIENT_SECRET", ""),
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
