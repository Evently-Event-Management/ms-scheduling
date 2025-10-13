package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"ms-scheduling/internal/config"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

// UserInfo represents the user information returned from Keycloak
type UserInfo struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	Enabled       bool   `json:"enabled"`
	EmailVerified bool   `json:"emailVerified"`
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

// GetUserEmailByID retrieves a user's email from Keycloak using their ID.
// It uses the client credentials grant flow to authenticate the request.
// Note: The client must have the "view-users" role from realm-management client.
func GetUserEmailByID(cfg config.Config, client *http.Client, userID string) (string, error) {
	// First, get an access token
	token, err := GetM2MToken(cfg, client)
	if err != nil {
		log.Printf("Failed to get M2M token: %v", err)
		return "", fmt.Errorf("failed to get M2M token: %w", err)
	}

	// Construct the URL to the Keycloak Admin REST API user endpoint
	// Make sure the client has the realm-management view-users role assigned
	userURL := fmt.Sprintf("%s/admin/realms/%s/users/%s", cfg.KeycloakURL, cfg.KeycloakRealm, userID)
	log.Printf("Requesting user info from: %s", userURL)

	// Create the request
	req, err := http.NewRequest("GET", userURL, nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set the authorization header with the token
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP request to Keycloak failed: %v", err)
		return "", fmt.Errorf("HTTP request to Keycloak failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("Error closing response body: %v", cerr)
		}
	}()

	// Check the response status
	log.Printf("Keycloak user info response status: %s", resp.Status)
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Keycloak user info response body: %s", string(bodyBytes))
		return "", fmt.Errorf("failed to get user info, status: %s", resp.Status)
	}

	// Parse the response
	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		log.Printf("Error decoding user info response: %v", err)
		return "", fmt.Errorf("error decoding user info response: %w", err)
	}

	// Check if email exists
	if userInfo.Email == "" {
		return "", fmt.Errorf("email not found for user ID: %s", userID)
	}

	log.Printf("Retrieved email for user ID %s: %s", userID, userInfo.Email)
	return userInfo.Email, nil
}

// ExtractTokenFromRequest extracts the bearer token from an HTTP request
func ExtractTokenFromRequest(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header is missing")
	}

	// Bearer token format: "Bearer {token}"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("authorization header format must be 'Bearer {token}'")
	}

	return parts[1], nil
}

// ExtractUserIDFromJWT extracts the user ID from a JWT token
// This function parses the JWT and extracts the 'sub' claim which contains the user ID
func ExtractUserIDFromJWT(tokenString string) (string, error) {
	if tokenString == "" {
		return "", errors.New("empty token")
	}

	// Parse the JWT without validating the signature
	// In a production environment, you should validate the signature
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	// Extract claims from token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token claims")
	}

	// Extract the subject claim which contains the user ID
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", errors.New("subject claim not found in token")
	}

	return sub, nil
}
