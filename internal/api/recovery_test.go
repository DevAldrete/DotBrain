package api

import (
	"context"
	"strings"
	"testing"
	"time"

	db "github.com/devaldrete/dotbrain/internal/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

// TestRecoverStaleRuns_MarksRunningAsFailed verifies that RecoverStaleRuns
// calls FailStaleRuns with the correct error message.
func TestRecoverStaleRuns_MarksRunningAsFailed(t *testing.T) {
	recorder := &queryRecorder{}
	queries := db.New(recorder)
	api := &API{queries: queries, activeRuns: newActiveRunRegistry()}

	err := api.RecoverStaleRuns(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify FailStaleRuns was called
	call := recorder.findCall("UPDATE workflow_runs")
	if call == nil {
		t.Fatal("expected FailStaleRuns query to be called")
	}

	// Verify the SQL targets only running/pending statuses
	if !strings.Contains(call.query, "status IN ('running', 'pending')") {
		t.Errorf("expected query to filter by running/pending statuses, got: %s", call.query)
	}

	// Verify the error message is set
	errArg, ok := call.args[0].(pgtype.Text)
	if !ok {
		t.Fatalf("expected first arg to be pgtype.Text, got %T", call.args[0])
	}
	if !errArg.Valid {
		t.Error("expected error message to be non-null")
	}
	if !strings.Contains(errArg.String, "server restarted") {
		t.Errorf("expected error message to mention 'server restarted', got %q", errArg.String)
	}
}

// TestRecoverStaleRuns_DoesNotTouchCompletedRuns verifies that the SQL query
// only targets running/pending runs, not completed ones.
func TestRecoverStaleRuns_DoesNotTouchCompletedRuns(t *testing.T) {
	recorder := &queryRecorder{}
	queries := db.New(recorder)
	api := &API{queries: queries, activeRuns: newActiveRunRegistry()}

	err := api.RecoverStaleRuns(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The FailStaleRuns query should only match running/pending — never completed.
	// Verify by checking the SQL WHERE clause does NOT include 'completed'.
	call := recorder.findCall("UPDATE workflow_runs")
	if call == nil {
		t.Fatal("expected FailStaleRuns query to be called")
	}

	if strings.Contains(call.query, "'completed'") {
		t.Error("FailStaleRuns query must not target completed runs")
	}
	// Extract the WHERE clause to verify it doesn't match already-failed runs.
	// The SET clause correctly uses 'failed' — we only care about the WHERE.
	whereIdx := strings.Index(call.query, "WHERE")
	if whereIdx == -1 {
		t.Fatal("expected WHERE clause in FailStaleRuns query")
	}
	whereClause := call.query[whereIdx:]
	if strings.Contains(whereClause, "'failed'") {
		t.Error("FailStaleRuns WHERE clause must not target already-failed runs")
	}
}

// TestWatchdog_TimesOutLongRunningRun verifies that FailRunsExceedingDuration
// is called with the correct threshold timestamp.
func TestWatchdog_TimesOutLongRunningRun(t *testing.T) {
	recorder := &queryRecorder{}
	queries := db.New(recorder)
	api := &API{queries: queries, activeRuns: newActiveRunRegistry()}

	maxDuration := 1 * time.Hour
	before := time.Now()

	err := api.FailTimedOutRuns(context.Background(), maxDuration)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after := time.Now()

	// Verify FailRunsExceedingDuration was called
	call := recorder.findCall("UPDATE workflow_runs")
	if call == nil {
		t.Fatal("expected FailRunsExceedingDuration query to be called")
	}

	// Verify the SQL targets only running status with started_at check
	if !strings.Contains(call.query, "status = 'running'") {
		t.Errorf("expected query to filter by running status, got: %s", call.query)
	}
	if !strings.Contains(call.query, "started_at") {
		t.Errorf("expected query to check started_at, got: %s", call.query)
	}

	// Verify the threshold timestamp is approximately now - maxDuration
	thresholdArg, ok := call.args[1].(pgtype.Timestamptz)
	if !ok {
		t.Fatalf("expected second arg to be pgtype.Timestamptz, got %T", call.args[1])
	}
	if !thresholdArg.Valid {
		t.Fatal("expected threshold timestamp to be valid")
	}

	expectedThreshold := before.Add(-maxDuration)
	// Allow 1 second tolerance for test execution time
	if thresholdArg.Time.Before(expectedThreshold.Add(-1*time.Second)) ||
		thresholdArg.Time.After(after.Add(-maxDuration).Add(1*time.Second)) {
		t.Errorf("threshold time %v not within expected range [%v, %v]",
			thresholdArg.Time, expectedThreshold.Add(-1*time.Second), after.Add(-maxDuration).Add(1*time.Second))
	}

	// Verify the error message mentions timeout
	errArg, ok := call.args[0].(pgtype.Text)
	if !ok {
		t.Fatalf("expected first arg to be pgtype.Text, got %T", call.args[0])
	}
	if !strings.Contains(errArg.String, "timed out") {
		t.Errorf("expected error message to mention 'timed out', got %q", errArg.String)
	}
}
