package api

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	db "github.com/devaldrete/dotbrain/internal/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

// RecoverStaleRuns marks any workflow_runs stuck in "running" or "pending"
// as "failed". This handles the case where the server was restarted while
// runs were in-flight.
func (a *API) RecoverStaleRuns(ctx context.Context) error {
	var errMsg pgtype.Text
	errMsg.String = "run aborted: server restarted while execution was in progress"
	errMsg.Valid = true

	count, err := a.queries.FailStaleRuns(ctx, errMsg)
	if err != nil {
		return fmt.Errorf("failed to recover stale runs: %w", err)
	}
	if count > 0 {
		slog.Warn("recovered stale runs", "count", count)
	}
	return nil
}

// FailTimedOutRuns marks runs that have been in "running" state longer
// than maxDuration as "failed". Called by the watchdog goroutine.
func (a *API) FailTimedOutRuns(ctx context.Context, maxDuration time.Duration) error {
	threshold := time.Now().Add(-maxDuration)

	var pgThreshold pgtype.Timestamptz
	pgThreshold.Time = threshold
	pgThreshold.Valid = true

	var errMsg pgtype.Text
	errMsg.String = fmt.Sprintf("run timed out: exceeded maximum duration of %s", maxDuration)
	errMsg.Valid = true

	count, err := a.queries.FailRunsExceedingDuration(ctx, db.FailRunsExceedingDurationParams{
		Error:     errMsg,
		StartedAt: pgThreshold,
	})
	if err != nil {
		return fmt.Errorf("watchdog query failed: %w", err)
	}
	if count > 0 {
		slog.Warn("watchdog timed out stale runs", "count", count)
	}
	return nil
}

// RunWatchdog periodically scans for runs that have been in "running" state
// longer than maxDuration and marks them failed.
func (a *API) RunWatchdog(ctx context.Context, maxDuration time.Duration, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.FailTimedOutRuns(ctx, maxDuration); err != nil {
				slog.Error("watchdog query failed", "error", err)
			}
		}
	}
}
