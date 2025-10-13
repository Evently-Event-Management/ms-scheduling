package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type KeycloakClient struct {
	BaseURL      string
	Realm        string
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
}

type KeycloakTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type KeycloakUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// KeycloakUserDetails represents extended user information from Keycloak
type KeycloakUserDetails struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func NewKeycloakClient(baseURL, realm, clientID, clientSecret string) *KeycloakClient {
	return &KeycloakClient{
		BaseURL:      baseURL,
		Realm:        realm,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// GetUserEmail fetches user email from Keycloak by UserID
func (k *KeycloakClient) GetUserEmail(userID string) (string, error) {
	// Get admin token
	token, err := k.getAdminToken()
	if err != nil {
		return "", fmt.Errorf("failed to get admin token: %v", err)
	}

	// Get user details
	url := fmt.Sprintf("%s/admin/realms/%s/users/%s", k.BaseURL, k.Realm, userID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := k.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("keycloak API error: %d - %s", resp.StatusCode, string(body))
	}

	var user KeycloakUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", err
	}

	if user.Email == "" {
		return "", fmt.Errorf("user %s has no email address", userID)
	}

	return user.Email, nil
}

// getAdminToken gets an admin token for Keycloak API calls
func (k *KeycloakClient) getAdminToken() (string, error) {
	url := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", k.BaseURL, k.Realm)

	data := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s",
		k.ClientID, k.ClientSecret)

	req, err := http.NewRequest("POST", url, bytes.NewBufferString(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := k.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed: %d - %s", resp.StatusCode, string(body))
	}

	var tokenResp KeycloakTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

// GetUserDetails fetches extended user information from Keycloak by UserID
func (k *KeycloakClient) GetUserDetails(userID string) (*KeycloakUserDetails, error) {
	// Get admin token
	token, err := k.getAdminToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %v", err)
	}

	// Get user details
	url := fmt.Sprintf("%s/admin/realms/%s/users/%s", k.BaseURL, k.Realm, userID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := k.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("keycloak API error: %d - %s", resp.StatusCode, string(body))
	}

	var userDetails KeycloakUserDetails
	if err := json.NewDecoder(resp.Body).Decode(&userDetails); err != nil {
		return nil, err
	}

	return &userDetails, nil
}
