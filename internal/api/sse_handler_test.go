package api

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	db "github.com/devaldrete/dotbrain/internal/db/sqlc"
)

// ---------------------------------------------------------------------------
// SSE handler tests
// ---------------------------------------------------------------------------

// TestStreamRunHandler_ReturnsSSEContentType checks that the endpoint responds
// with the correct Content-Type for Server-Sent Events.
func TestStreamRunHandler_ReturnsSSEContentType(t *testing.T) {
	bus := NewEventBus()
	runIDStr := "01900000-0000-7000-8000-000000000010"

	api := &API{
		queries:    db.New(&queryRecorder{}),
		activeRuns: newActiveRunRegistry(),
		bus:        bus,
	}
	router := api.NewRouter("")

	server := httptest.NewServer(router)
	defer server.Close()

	// Use a context with timeout so the SSE connection is closed after headers arrive
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/v1/runs/"+runIDStr+"/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil && ctx.Err() == nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp != nil {
		defer resp.Body.Close()
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
			t.Errorf("expected Content-Type text/event-stream, got %q", ct)
		}
	}
}

// TestStreamRunHandler_StreamsPublishedEvents verifies that events published
// to the bus are forwarded as SSE frames to the client.
func TestStreamRunHandler_StreamsPublishedEvents(t *testing.T) {
	bus := NewEventBus()
	runIDStr := "01900000-0000-7000-8000-000000000011"

	api := &API{
		queries:    db.New(&queryRecorder{}),
		activeRuns: newActiveRunRegistry(),
		bus:        bus,
	}
	router := api.NewRouter("")

	server := httptest.NewServer(router)
	defer server.Close()

	// Channel to receive scan results from the reading goroutine
	found := make(chan bool, 1)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/v1/runs/"+runIDStr+"/stream", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			found <- false
			return
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "node.started") {
				found <- true
				cancel()
				return
			}
		}
		found <- false
	}()

	// Give the handler time to subscribe on the server side
	time.Sleep(50 * time.Millisecond)

	// Publish the event
	bus.Publish(runIDStr, RunEvent{Type: "node.started", RunID: runIDStr, NodeID: "step-1"})

	select {
	case got := <-found:
		if !got {
			t.Error("expected node.started event in SSE stream")
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for node.started event")
	}
}

// TestStreamRunHandler_TerminalRunEmitsImmediately verifies that if the run is
// already in a terminal state when the client connects, a terminal event is
// emitted and the stream closes.
func TestStreamRunHandler_TerminalRunEmitsImmediately(t *testing.T) {
	bus := NewEventBus()
	runIDStr := "01900000-0000-7000-8000-000000000012"

	qr := &queryRecorder{
		runStatus: "completed",
		runIDStr:  runIDStr,
	}

	api := &API{
		queries:    db.New(qr),
		activeRuns: newActiveRunRegistry(),
		bus:        bus,
	}
	router := api.NewRouter("")

	server := httptest.NewServer(router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/runs/" + runIDStr + "/stream")
	if err != nil {
		t.Fatalf("SSE request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read lines until we find run.completed or timeout
	scanner := bufio.NewScanner(resp.Body)
	found := false
	done := make(chan struct{})
	go func() {
		defer close(done)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "run.completed") {
				found = true
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
	}

	if !found {
		t.Error("expected run.completed terminal event for already-completed run")
	}
}

// TestStreamRunHandler_InvalidRunID returns 400 for a non-UUID path parameter.
func TestStreamRunHandler_InvalidRunID(t *testing.T) {
	bus := NewEventBus()
	api := &API{
		queries:    db.New(&queryRecorder{}),
		activeRuns: newActiveRunRegistry(),
		bus:        bus,
	}
	router := api.NewRouter("")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/runs/not-a-uuid/stream", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid run ID, got %d", w.Code)
	}
}
