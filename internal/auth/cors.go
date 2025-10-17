package auth

import (
	"net/http"
	"strconv"
	"strings"

	"ms-scheduling/internal/config"
)

// CORSMiddleware adds CORS headers to responses based on configuration
func CORSMiddleware(cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if the origin is allowed
			allowedOrigin := ""
			for _, allowed := range cfg.AllowedOrigins {
				if allowed == "*" || allowed == origin {
					allowedOrigin = origin
					break
				}
			}

			// If we didn't find an exact match but we have wildcard domains
			if allowedOrigin == "" {
				for _, allowed := range cfg.AllowedOrigins {
					// Handle wildcard subdomains like *.example.com
					if strings.HasPrefix(allowed, "*.") && origin != "" {
						domain := allowed[1:] // remove the *
						if strings.HasSuffix(origin, domain) {
							allowedOrigin = origin
							break
						}
					}
				}
			}

			// Set CORS headers if origin is allowed
			if allowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))

				// Handle preflight requests
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusOK)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
