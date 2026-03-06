package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// RunEvent is a single event emitted during a workflow run lifecycle.
type RunEvent struct {
	Type   string         `json:"type"`
	RunID  string         `json:"run_id,omitempty"`
	NodeID string         `json:"node_id,omitempty"`
	Data   map[string]any `json:"data,omitempty"`
}

// EventBus is an in-memory pub/sub bus keyed by run UUID.
// Each subscriber gets its own buffered channel so a slow consumer
// cannot block the publisher.
type EventBus struct {
	mu   sync.Mutex
	subs map[string][]chan RunEvent
}

// NewEventBus returns an initialised EventBus.
func NewEventBus() *EventBus {
	return &EventBus{
		subs: make(map[string][]chan RunEvent),
	}
}

// Subscribe registers a new subscriber for the given run ID and returns
// a receive-only channel that will receive all future events for that run.
func (b *EventBus) Subscribe(runID string) <-chan RunEvent {
	ch := make(chan RunEvent, 64) // buffered to tolerate brief slow consumers
	b.mu.Lock()
	b.subs[runID] = append(b.subs[runID], ch)
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes the subscriber channel for the given run ID and
// closes it so the consumer's range loop terminates cleanly.
func (b *EventBus) Unsubscribe(runID string, ch <-chan RunEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	list := b.subs[runID]
	for i, c := range list {
		if c == ch {
			// Remove from slice and close
			b.subs[runID] = append(list[:i], list[i+1:]...)
			close(c)
			break
		}
	}
	if len(b.subs[runID]) == 0 {
		delete(b.subs, runID)
	}
}

// Publish sends an event to all current subscribers of the given run ID.
// Non-blocking: if a subscriber's buffer is full the event is dropped for
// that subscriber rather than blocking the publisher goroutine.
// Safe to call on a nil *EventBus (no-op).
func (b *EventBus) Publish(runID string, event RunEvent) {
	if b == nil {
		return
	}
	b.mu.Lock()
	list := make([]chan RunEvent, len(b.subs[runID]))
	copy(list, b.subs[runID])
	b.mu.Unlock()

	for _, ch := range list {
		select {
		case ch <- event:
		default:
			// subscriber too slow — drop event for this subscriber
		}
	}
}

// terminalStatuses is the set of run statuses that close the SSE stream.
var terminalStatuses = map[string]bool{
	"completed": true,
	"failed":    true,
	"cancelled": true,
}

// terminalEventType maps a terminal run status to the SSE event type string.
func terminalEventType(status string) string {
	switch status {
	case "completed":
		return "run.completed"
	case "failed":
		return "run.failed"
	case "cancelled":
		return "run.cancelled"
	default:
		return "run." + status
	}
}

// streamRunHandler handles GET /api/v1/runs/:id/stream.
// It opens an SSE connection and streams RunEvents from the EventBus until
// a terminal event is received or the client disconnects.
func (a *API) streamRunHandler(c *gin.Context) {
	idStr := c.Param("id")
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run ID"})
		return
	}

	var pgID pgtype.UUID
	pgID.Bytes = parsedID
	pgID.Valid = true

	// Fetch current run status. If already terminal, emit the event immediately
	// and close. This handles the case where the client connects after the run
	// has already finished.
	if a.queries != nil {
		run, err := a.queries.GetWorkflowRun(c.Request.Context(), pgID)
		if err == nil && terminalStatuses[run.Status] {
			data, _ := json.Marshal(map[string]string{"run_id": idStr, "status": run.Status})
			c.Header("Content-Type", "text/event-stream")
			c.Header("Cache-Control", "no-cache")
			c.Header("X-Accel-Buffering", "no")
			fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", terminalEventType(run.Status), data)
			c.Writer.(http.Flusher).Flush()
			return
		}
	}

	// Subscribe to future events for this run
	ch := a.bus.Subscribe(idStr)
	defer a.bus.Unsubscribe(idStr, ch)

	clientGone := c.Request.Context().Done()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-clientGone:
			return false
		case evt, ok := <-ch:
			if !ok {
				return false
			}
			data, _ := json.Marshal(evt)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, data)
			// Flush is handled by gin's c.Stream
			if terminalStatuses[evt.Type] {
				return false // close stream after terminal event
			}
			return true
		}
	})
}
