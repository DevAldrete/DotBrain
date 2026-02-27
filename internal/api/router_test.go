package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheckHandler(t *testing.T) {
	router := NewRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/health", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["status"] != "UP" {
		t.Errorf("expected status 'UP', got '%s'", response["status"])
	}

	if response["timestamp"] == "" {
		t.Errorf("expected a timestamp, got empty string")
	}
}

func TestReadinessHandler(t *testing.T) {
	router := NewRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/readiness", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["status"] != "READY" {
		t.Errorf("expected status 'READY', got '%s'", response["status"])
	}
}

func TestPingHandler(t *testing.T) {
	router := NewRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/ping", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["message"] != "pong" {
		t.Errorf("expected message 'pong', got '%s'", response["message"])
	}
}

func TestWorkflowTriggerHandler_UnknownID(t *testing.T) {
	router := NewRouter()

	w := httptest.NewRecorder()
	reqBody, _ := json.Marshal(map[string]any{"data": "test"})
	req, _ := http.NewRequest("POST", "/api/v1/workflows/unknown-id/trigger", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestWorkflowTriggerHandler_Success(t *testing.T) {
	router := NewRouter()

	w := httptest.NewRecorder()
	reqBody, _ := json.Marshal(map[string]any{"data": "test"})
	// Assume "valid-id" is a known ID for the mock
	req, _ := http.NewRequest("POST", "/api/v1/workflows/valid-id/trigger", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, w.Code)
	}
}
