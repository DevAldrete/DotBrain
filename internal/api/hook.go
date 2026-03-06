package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/devaldrete/dotbrain/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// DBNodeHook implements core.NodeLifecycleHook to record node executions in the database.
type DBNodeHook struct {
	queries    *db.Queries
	runID      pgtype.UUID
	runIDStr   string    // string form used as EventBus key
	bus        *EventBus // may be nil if streaming is not configured
	executions map[string]pgtype.UUID
	startTimes map[string]time.Time
}

// NewDBNodeHook creates a new DBNodeHook without an EventBus (backwards-compatible).
func NewDBNodeHook(queries *db.Queries, runID pgtype.UUID) *DBNodeHook {
	return &DBNodeHook{
		queries:    queries,
		runID:      runID,
		executions: make(map[string]pgtype.UUID),
		startTimes: make(map[string]time.Time),
	}
}

// NewDBNodeHookWithBus creates a DBNodeHook that also publishes lifecycle events
// to the provided EventBus so SSE subscribers receive real-time updates.
func NewDBNodeHookWithBus(queries *db.Queries, runID pgtype.UUID, runIDStr string, bus *EventBus) *DBNodeHook {
	return &DBNodeHook{
		queries:    queries,
		runID:      runID,
		runIDStr:   runIDStr,
		bus:        bus,
		executions: make(map[string]pgtype.UUID),
		startTimes: make(map[string]time.Time),
	}
}

// publish sends an event to the EventBus if one is configured.
func (h *DBNodeHook) publish(evt RunEvent) {
	if h.bus != nil && h.runIDStr != "" {
		h.bus.Publish(h.runIDStr, evt)
	}
}

// OnNodeStart records the start of a node execution.
func (h *DBNodeHook) OnNodeStart(ctx context.Context, nodeID string, input map[string]any) {
	executionID, err := uuid.NewV7()
	if err != nil {
		return // Should not fail, but if it does we skip DB recording
	}

	var pgExecutionID pgtype.UUID
	pgExecutionID.Bytes = executionID
	pgExecutionID.Valid = true

	h.executions[nodeID] = pgExecutionID
	h.startTimes[nodeID] = time.Now()

	var inputBytes []byte
	if input != nil {
		inputBytes, _ = json.Marshal(input)
	}

	_, _ = h.queries.CreateNodeExecution(ctx, db.CreateNodeExecutionParams{
		ID:            pgExecutionID,
		WorkflowRunID: h.runID,
		NodeID:        nodeID,
		Status:        "running",
		InputData:     inputBytes,
	})

	h.publish(RunEvent{Type: "node.started", RunID: h.runIDStr, NodeID: nodeID, Data: input})
}

// OnNodeComplete records the successful completion of a node.
func (h *DBNodeHook) OnNodeComplete(ctx context.Context, nodeID string, output map[string]any) {
	executionID, ok := h.executions[nodeID]
	if !ok {
		return
	}

	startedAt := h.startTimes[nodeID]
	var pgStartedAt pgtype.Timestamptz
	pgStartedAt.Time = startedAt
	pgStartedAt.Valid = true

	var pgCompletedAt pgtype.Timestamptz
	pgCompletedAt.Time = time.Now()
	pgCompletedAt.Valid = true

	var outputBytes []byte
	if output != nil {
		outputBytes, _ = json.Marshal(output)
	}

	_, _ = h.queries.UpdateNodeExecutionStatus(ctx, db.UpdateNodeExecutionStatusParams{
		ID:          executionID,
		Status:      "completed",
		OutputData:  outputBytes,
		StartedAt:   pgStartedAt,
		CompletedAt: pgCompletedAt,
	})

	h.publish(RunEvent{Type: "node.completed", RunID: h.runIDStr, NodeID: nodeID, Data: output})
}

// OnNodeFail records the failure of a node.
func (h *DBNodeHook) OnNodeFail(ctx context.Context, nodeID string, err error) {
	executionID, ok := h.executions[nodeID]
	if !ok {
		return
	}

	startedAt := h.startTimes[nodeID]
	var pgStartedAt pgtype.Timestamptz
	pgStartedAt.Time = startedAt
	pgStartedAt.Valid = true

	var pgCompletedAt pgtype.Timestamptz
	pgCompletedAt.Time = time.Now()
	pgCompletedAt.Valid = true

	var pgErr pgtype.Text
	if err != nil {
		pgErr.String = err.Error()
		pgErr.Valid = true
	}

	_, _ = h.queries.UpdateNodeExecutionStatus(ctx, db.UpdateNodeExecutionStatusParams{
		ID:          executionID,
		Status:      "failed",
		Error:       pgErr,
		StartedAt:   pgStartedAt,
		CompletedAt: pgCompletedAt,
	})

	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	h.publish(RunEvent{Type: "node.failed", RunID: h.runIDStr, NodeID: nodeID, Data: map[string]any{"error": errMsg}})
}

// OnNodeRetry records that a node is being retried.
func (h *DBNodeHook) OnNodeRetry(ctx context.Context, nodeID string, attempt int, err error) {
	executionID, ok := h.executions[nodeID]
	if !ok {
		return
	}

	var pgErr pgtype.Text
	if err != nil {
		pgErr.String = fmt.Sprintf("attempt %d: %s", attempt, err.Error())
		pgErr.Valid = true
	}

	_, _ = h.queries.UpdateNodeExecutionStatus(ctx, db.UpdateNodeExecutionStatusParams{
		ID:     executionID,
		Status: "retrying",
		Error:  pgErr,
	})

	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	h.publish(RunEvent{Type: "node.retrying", RunID: h.runIDStr, NodeID: nodeID, Data: map[string]any{"attempt": attempt, "error": errMsg}})
}
