package session

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"ms-scheduling/internal/config"
	"ms-scheduling/internal/models"
)

// ProcessSessionMessage makes the API call to the Event Service to update the session status or sends reminder emails.
func ProcessSessionMessage(cfg config.Config, client *http.Client, token string, msg *models.SQSMessageBody, subscriberService interface{}) error {
	var apiPath string

	switch msg.Action {
	case "ON_SALE":
		apiPath = fmt.Sprintf("/internal/v1/sessions/%s/on-sale", msg.SessionID)
	case "CLOSED":
		apiPath = fmt.Sprintf("/internal/v1/sessions/%s/closed", msg.SessionID)
	case "REMINDER_EMAIL":
		// Handle reminder email - this doesn't call the Event Service API
		return ProcessReminderEmail(msg.SessionID, subscriberService)
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

		if resp.StatusCode == http.StatusConflict {
			log.Printf("Session %s is in a conflicting state (409). Treating as successfully processed to avoid infinite retries.", msg.SessionID)
			return nil
		}

		return fmt.Errorf("API call failed with status %s: %s", resp.Status, string(bodyBytes))
	}

	log.Printf("Successfully processed action '%s' for session %s", msg.Action, msg.SessionID)
	return nil
}

// ProcessReminderEmail handles the reminder email action
func ProcessReminderEmail(sessionID string, subscriberService interface{}) error {
	log.Printf("Processing reminder email for session %s", sessionID)

	// Type assert the subscriber service to access the ProcessSessionReminder method
	if ss, ok := subscriberService.(interface {
		ProcessSessionReminder(string) error
	}); ok {
		err := ss.ProcessSessionReminder(sessionID)
		if err != nil {
			log.Printf("Error sending reminder emails for session %s: %v", sessionID, err)
			return err
		}
		log.Printf("Successfully sent reminder emails for session %s", sessionID)
		return nil
	}

	return fmt.Errorf("subscriber service does not implement ProcessSessionReminder method")
}
