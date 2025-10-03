package session

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"ms-scheduling/internal/config"
	"ms-scheduling/internal/models"
)

// ProcessSessionMessage makes the API call to the Event Service to update the session status.
func ProcessSessionMessage(cfg config.Config, client *http.Client, token string, msg *models.SQSMessageBody) error {
	var apiPath string

	switch msg.Action {
	case "ON_SALE":
		apiPath = fmt.Sprintf("/internal/v1/sessions/%s/on-sale", msg.SessionID)
	case "CLOSED":
		apiPath = fmt.Sprintf("/internal/v1/sessions/%s/closed", msg.SessionID)
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

		// Special handling for 404 errors - if the session is not found, we consider the message processed
		// This prevents an infinite loop of retrying non-existent sessions
		if resp.StatusCode == http.StatusNotFound {
			log.Printf("Session %s not found (404). Treating as successfully processed to avoid infinite retries.", msg.SessionID)
			return nil
		}

		return fmt.Errorf("API call failed with status %s: %s", resp.Status, string(bodyBytes))
	}

	log.Printf("Successfully processed action '%s' for session %s", msg.Action, msg.SessionID)
	return nil
}
