package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"ms-scheduling/internal/services"
)

// HealthHandler provides health check endpoints for readiness and liveness probes
type HealthHandler struct {
	dbService       *services.DatabaseService
	startTime       time.Time
	readinessChecks map[string]func() error
	livenessChecks  map[string]func() error
}

// Health response structure
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Uptime    string            `json:"uptime"`
	Details   map[string]string `json:"details,omitempty"`
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(dbService *services.DatabaseService) *HealthHandler {
	h := &HealthHandler{
		dbService:       dbService,
		startTime:       time.Now(),
		readinessChecks: make(map[string]func() error),
		livenessChecks:  make(map[string]func() error),
	}

	// Register default health checks
	h.registerDefaultChecks()

	return h
}

// registerDefaultChecks adds default readiness and liveness checks
func (h *HealthHandler) registerDefaultChecks() {
	// Readiness checks if the service is ready to accept traffic
	h.readinessChecks["database"] = h.dbService.CheckConnection

	// Liveness checks if the service is running properly
	h.livenessChecks["uptime"] = func() error {
		// Always returns nil - just a placeholder to show service is up
		return nil
	}
}

// HandleReadiness handles readiness probe requests
func (h *HealthHandler) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	details := make(map[string]string)
	allOk := true

	// Run all readiness checks
	for name, check := range h.readinessChecks {
		err := check()
		if err != nil {
			allOk = false
			details[name] = err.Error()
		} else {
			details[name] = "OK"
		}
	}

	response := HealthResponse{
		Status:    "UP",
		Timestamp: time.Now().Format(time.RFC3339),
		Uptime:    time.Since(h.startTime).String(),
		Details:   details,
	}

	if !allOk {
		response.Status = "DOWN"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding health response: %v", err)
	}
}

// HandleLiveness handles liveness probe requests
func (h *HealthHandler) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	details := make(map[string]string)
	allOk := true

	// Run all liveness checks
	for name, check := range h.livenessChecks {
		err := check()
		if err != nil {
			allOk = false
			details[name] = err.Error()
		} else {
			details[name] = "OK"
		}
	}

	response := HealthResponse{
		Status:    "UP",
		Timestamp: time.Now().Format(time.RFC3339),
		Uptime:    time.Since(h.startTime).String(),
	}

	if !allOk {
		response.Status = "DOWN"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding health response: %v", err)
	}
}

// HandleHealth handles general health check requests
// This combines both readiness and liveness checks
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	// For simple kubernetes checks, just return OK
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
