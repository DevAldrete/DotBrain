# TASK-11 — Crash Recovery for Stale Running Runs

**Phase:** 5 — Engine Evolution  
**Priority:** Critical  
**Depends on:** nothing  
**Files affected:** `internal/api/router.go` (or new `internal/api/recovery.go`), `cmd/dotbrain/main.go`, `query.sql`, `internal/db/sqlc/`

---

## Problem

Workflow runs execute inside goroutines with no crash protection or timeout. If the server process is killed, restarted (e.g., by Kubernetes rolling update, OOM kill, or `air` reload during development), any in-flight runs are left permanently in `status = 'running'`.

There is no mechanism to detect or recover from this. A caller polling `GET /api/v1/runs/:id` will wait forever. The run history is permanently misleading. Re-triggering the workflow creates a new run, leaving the ghost run in `running` indefinitely.

This is a correctness issue, not just a quality-of-life issue.

---

## Goal

1. **Startup recovery:** On server start, scan for any `workflow_runs` stuck in `running` or `pending` and mark them `failed` with a clear error message.
2. **Run timeout watchdog:** A background goroutine periodically marks runs that have been `running` for longer than a configurable max duration as `failed`.

---

## Implementation

### Part 1 — Startup recovery

Add a `RecoverStaleRuns` function called during application startup, before the HTTP server begins accepting requests:

```go
// internal/api/recovery.go

// RecoverStaleRuns marks any workflow_runs stuck in "running" or "pending"
// as "failed". This handles the case where the server was restarted while
// runs were in-flight.
func (a *API) RecoverStaleRuns(ctx context.Context) error {
    count, err := a.queries.FailStaleRuns(ctx, db.FailStaleRunsParams{
        Statuses: []string{"running", "pending"},
        ErrorMsg: "run aborted: server restarted while execution was in progress",
    })
    if err != nil {
        return fmt.Errorf("failed to recover stale runs: %w", err)
    }
    if count > 0 {
        slog.Warn("recovered stale runs", "count", count)
    }
    return nil
}
```

Called in `cmd/dotbrain/main.go` before `router.Run(...)`:

```go
if err := api.RecoverStaleRuns(ctx); err != nil {
    slog.Error("startup recovery failed", "error", err)
    // non-fatal: continue starting up
}
```

### Part 2 — Watchdog goroutine

```go
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
            threshold := time.Now().Add(-maxDuration)
            count, err := a.queries.FailRunsExceedingDuration(ctx, db.FailRunsExceedingDurationParams{
                StartedBefore: threshold,
                ErrorMsg:      fmt.Sprintf("run timed out: exceeded maximum duration of %s", maxDuration),
            })
            if err != nil {
                slog.Error("watchdog query failed", "error", err)
                continue
            }
            if count > 0 {
                slog.Warn("watchdog timed out stale runs", "count", count)
            }
        }
    }
}
```

Started in `main.go` as a background goroutine, shut down via context cancellation on SIGTERM.

### New SQL Queries

```sql
-- name: FailStaleRuns :execrows
UPDATE workflow_runs
SET status = 'failed',
    error = @error_msg,
    completed_at = NOW()
WHERE status = ANY(@statuses::text[]);

-- name: FailRunsExceedingDuration :execrows
UPDATE workflow_runs
SET status = 'failed',
    error = @error_msg,
    completed_at = NOW()
WHERE status = 'running'
  AND started_at < @started_before;
```

---

## Configuration

New environment variables:

| Variable | Default | Description |
|---|---|---|
| `RUN_MAX_DURATION` | `1h` | Runs exceeding this duration are killed by the watchdog |
| `WATCHDOG_INTERVAL` | `5m` | How often the watchdog scans |

---

## Acceptance Criteria

- [ ] On startup, all `workflow_runs` with `status IN ('running', 'pending')` are immediately transitioned to `failed`
- [ ] The `error` column is set to a human-readable message indicating a server restart
- [ ] `completed_at` is set on recovered rows
- [ ] A log line is emitted at `WARN` level for each recovered run (or a count)
- [ ] The watchdog goroutine starts after startup recovery
- [ ] Runs that exceed `RUN_MAX_DURATION` are transitioned to `failed` by the watchdog
- [ ] Stopping the server (SIGTERM) stops the watchdog goroutine cleanly via context cancellation
- [ ] `go test ./...` passes; add tests for `RecoverStaleRuns` using a test DB or mock

---

## TDD Approach

### Red

```go
// TestRecoverStaleRuns_MarksRunningAsFailed
// Insert a workflow_run with status='running' into test DB.
// Call RecoverStaleRuns. Assert status is now 'failed' with correct error message.
func TestRecoverStaleRuns_MarksRunningAsFailed(t *testing.T) { ... }

// TestRecoverStaleRuns_DoesNotTouchCompletedRuns
// Insert a workflow_run with status='completed'. Call RecoverStaleRuns.
// Assert status is still 'completed'.
func TestRecoverStaleRuns_DoesNotTouchCompletedRuns(t *testing.T) { ... }

// TestWatchdog_TimesOutLongRunningRun
// Insert a workflow_run with status='running', started_at = 2 hours ago.
// Call watchdog logic with maxDuration=1h. Assert status becomes 'failed'.
func TestWatchdog_TimesOutLongRunningRun(t *testing.T) { ... }
```

---

## Definition of Done

- All acceptance criteria checked
- `go test ./...` passes with no regressions
- Startup log clearly indicates how many stale runs were recovered
- `docs/core/data-model.md` updated: note that `running`/`pending` runs are cleaned on startup
