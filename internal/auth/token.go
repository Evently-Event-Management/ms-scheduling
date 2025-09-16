package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ms-scheduling/internal/config"
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

// GetM2MToken performs the Client Credentials Grant flow to get a machine-to-machine token.
func GetM2MToken(cfg config.Config, client *http.Client) (string, error) {
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", cfg.KeycloakURL, cfg.KeycloakRealm)
	log.Printf("Requesting M2M token from: %s", tokenURL)

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", cfg.ClientID)
	data.Set("client_secret", cfg.ClientSecret)

	req, _ := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		// Do not mutate the passed client. Assume the caller sets timeout as needed.
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

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		log.Printf("Error decoding token response: %v", err)
		return "", err
	}
	log.Printf("Received access token: %s", tokenResp.AccessToken)

	return tokenResp.AccessToken, nil
}
