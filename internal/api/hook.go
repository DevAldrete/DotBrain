package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/devaldrete/dotbrain/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// DBNodeHook implements core.NodeLifecycleHook to record node executions in the database.
type DBNodeHook struct {
	queries     *db.Queries
	runID       pgtype.UUID
	executions  map[string]pgtype.UUID
	startTimes  map[string]time.Time
}

// NewDBNodeHook creates a new DBNodeHook.
func NewDBNodeHook(queries *db.Queries, runID pgtype.UUID) *DBNodeHook {
	return &DBNodeHook{
		queries:    queries,
		runID:      runID,
		executions: make(map[string]pgtype.UUID),
		startTimes: make(map[string]time.Time),
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
}
