package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
)

// User ID context key
type contextKey string

const (
	UserIDKey contextKey = "userID"
)

// GetUserIDFromContext extracts userID from context
func GetUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok || userID == "" {
		return "", errors.New("user ID not found in context")
	}
	return userID, nil
}

// HasAdminRole checks if the token has an admin role
// In a real implementation, this would parse and validate the JWT
// and check for admin roles in the claims
func HasAdminRole(token string) (bool, error) {
	// TODO: Implement proper JWT validation and role checking
	// For now, we'll just check if the token contains "admin" as a simple simulation
	// This is NOT secure and should be replaced with proper JWT validation
	return strings.Contains(strings.ToLower(token), "admin"), nil
}

// AuthMiddleware extracts user ID from the auth token and puts it in the request context
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from request
		token, err := ExtractTokenFromRequest(r)
		if err != nil {
			log.Printf("Error extracting token: %v", err)
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Extract user ID from token
		userID, err := ExtractUserIDFromJWT(token)
		if err != nil {
			log.Printf("Error extracting user ID from JWT: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		log.Printf("User authenticated with ID: %s", userID)

		// Add user ID to request context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AdminMiddleware checks if the user has admin role
func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from request
		token, err := ExtractTokenFromRequest(r)
		if err != nil {
			log.Printf("Error extracting token: %v", err)
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Check if user has admin role
		isAdmin, err := HasAdminRole(token)
		if err != nil {
			log.Printf("Error checking admin role: %v", err)
			http.Error(w, "Failed to validate authorization", http.StatusInternalServerError)
			return
		}

		if !isAdmin {
			http.Error(w, "Forbidden - Admin access required", http.StatusForbidden)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// extractSimulatedUserID extracts a user ID from a token for simulation
// This is NOT secure and should be replaced with proper JWT validation
func extractSimulatedUserID(token string) string {
	// In a real implementation, this would decode the JWT and extract the subject claim
	// For simulation, we'll use the first 8 characters of the token
	if len(token) > 8 {
		return token[:8]
	}
	return token
}
