# TASK-13 — Run Cancellation

**Phase:** 6 — API Completeness  
**Priority:** High  
**Depends on:** nothing  
**Files affected:** `internal/api/router.go`, `internal/api/runner.go` (new), `query.sql`, `internal/db/sqlc/`

---

## Problem

Once triggered, a workflow run cannot be stopped. Long-running nodes (slow external APIs, expensive LLM calls) execute to completion regardless. The `cancelled` status exists in the DB schema but nothing ever sets it.

This is a practical problem: a mistakenly triggered run with an expensive LLM node will burn API tokens with no way to stop it.

---

## Goal

Add `POST /api/v1/runs/:id/cancel` that:
1. Signals the in-flight goroutine to stop via context cancellation.
2. Marks the run as `cancelled` in the database.
3. Returns immediately; the goroutine cleans up asynchronously.

---

## Design: Active Run Registry

The core problem is that the HTTP handler and the execution goroutine are disconnected — there is no reference from `run_id` to the goroutine.

The solution is a **run registry**: a protected map in the `API` struct that stores a `context.CancelFunc` per active run.

```go
// internal/api/router.go (or new internal/api/runner.go)

type API struct {
    pool       *pgxpool.Pool
    queries    *db.Queries
    activeRuns activeRunRegistry
}

type activeRunRegistry struct {
    mu     sync.Mutex
    cancels map[string]context.CancelFunc  // run UUID string → cancel func
}

func (r *activeRunRegistry) register(runID string, cancel context.CancelFunc) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.cancels[runID] = cancel
}

func (r *activeRunRegistry) cancel(runID string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()
    cancel, ok := r.cancels[runID]
    if ok {
        cancel()
        delete(r.cancels, runID)
    }
    return ok
}

func (r *activeRunRegistry) deregister(runID string) {
    r.mu.Lock()
    defer r.mu.Unlock()
    delete(r.cancels, runID)
}
```

---

## Trigger Handler Changes

The trigger goroutine creates a **cancellable context** and registers it:

```go
// Before spawning the goroutine
runCtx, cancelRun := context.WithCancel(context.Background())
a.activeRuns.register(runID.String(), cancelRun)

go func(runID pgtype.UUID, ...) {
    defer a.activeRuns.deregister(runID.String())
    defer cancelRun()

    a.transitionToRunning(runCtx, runID)
    // ... engine setup ...
    output, err := engine.Execute(runCtx, initialData)

    if err != nil {
        if errors.Is(err, context.Canceled) {
            a.updateRunStatus(context.Background(), runID, "cancelled", nil, "run was cancelled")
        } else {
            a.updateRunStatus(context.Background(), runID, "failed", nil, err.Error())
        }
        return
    }
    a.updateRunStatus(context.Background(), runID, "completed", output, "")
}(pgRunID, workflow, payload)
```

Note: `updateRunStatus` uses `context.Background()` — not `runCtx` — so the final status update is not itself cancelled.

---

## New Endpoint

```go
// router.go
v1.POST("/runs/:id/cancel", a.cancelRunHandler)
```

```go
func (a *API) cancelRunHandler(c *gin.Context) {
    idStr := c.Param("id")
    if _, err := uuid.Parse(idStr); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run ID"})
        return
    }

    // Verify the run exists and is in a cancellable state
    parsedID, _ := uuid.Parse(idStr)
    var pgID pgtype.UUID
    pgID.Bytes = parsedID
    pgID.Valid = true

    run, err := a.queries.GetWorkflowRun(c, pgID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
        return
    }

    if run.Status != "pending" && run.Status != "running" {
        c.JSON(http.StatusConflict, gin.H{
            "error": fmt.Sprintf("run is already in terminal state: %s", run.Status),
        })
        return
    }

    // If the run is active in this process, cancel via context
    found := a.activeRuns.cancel(idStr)

    if !found {
        // Run is pending or was started on a different instance (future multi-process case).
        // Mark cancelled directly in the DB.
        a.updateRunStatus(c, pgID, "cancelled", nil, "run cancelled by user")
    }

    c.JSON(http.StatusAccepted, gin.H{"message": "cancellation requested"})
}
```

---

## New SQL Query

```sql
-- name: GetWorkflowRunStatus :one
SELECT status FROM workflow_runs WHERE id = $1;
```

(Or reuse `GetWorkflowRun` — the full row is small enough.)

---

## Acceptance Criteria

- [ ] `POST /api/v1/runs/:id/cancel` exists and returns 202
- [ ] Cancelling an active run causes the execution goroutine to stop
- [ ] After cancellation, `GET /api/v1/runs/:id` returns `status: "cancelled"`
- [ ] Cancelling a `completed` or `failed` run returns 409 Conflict
- [ ] Cancelling a non-existent run returns 404
- [ ] `context.Canceled` from the engine is correctly translated to `cancelled` status (not `failed`)
- [ ] Node I/O operations that respect context (HTTP requests, LLM calls) are interrupted promptly
- [ ] The active run registry has no goroutine leak — `deregister` is always called on run completion
- [ ] `go test ./internal/api/...` passes with tests for the cancel handler

---

## TDD Approach

```go
// TestCancelRun_ActiveRun — trigger a slow run, cancel it, verify status = cancelled
func TestCancelRun_ActiveRun(t *testing.T) { ... }

// TestCancelRun_AlreadyCompleted — assert 409 when run is completed
func TestCancelRun_AlreadyCompleted(t *testing.T) { ... }

// TestCancelRun_NotFound — assert 404 for unknown run ID
func TestCancelRun_NotFound(t *testing.T) { ... }

// TestActiveRunRegistry_NoLeak — register and deregister; verify map is empty
func TestActiveRunRegistry_NoLeak(t *testing.T) { ... }
```

---

## Definition of Done

- All acceptance criteria checked
- `go test ./...` passes with no regressions
- `docs/core/api.md` updated with the cancel endpoint
- `docs/core/data-model.md` updated: `cancelled` status is now reachable
