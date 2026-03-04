# TASK-04 — Fix Workflow Run Lifecycle (`pending → running`)

**Phase:** 1 — Make Execution Actually Work  
**Priority:** High (should be done before TASK-02)  
**Files affected:** `internal/api/router.go`, `query.sql`, `internal/db/sqlc/query.sql.go`

---

## Problem

The current trigger handler creates a `workflow_run` row with `status = "running"` immediately at request time, before the goroutine has even started:

```go
// internal/api/router.go (current)
_, err = a.queries.CreateWorkflowRun(c, db.CreateWorkflowRunParams{
    ID:         pgRunID,
    WorkflowID: pgID,
    Status:     "running",   // wrong — the goroutine hasn't started yet
    InputData:  inputBytes,
})
```

Additionally, `started_at` is never set. It remains `NULL` for the entire lifetime of every run, including completed ones.

The schema defines the intended lifecycle as:
```
pending → running → completed / failed / cancelled
```

But in practice:
- The `"pending"` state is never used
- `started_at` is always NULL

This means:
- Run duration cannot be calculated (`started_at` is NULL)
- There is no meaningful distinction between "queued but not started" and "actively executing"
- The `UpdateWorkflowRunStatus` query accepts a `started_at` param (via `sqlc.narg`) but it is never passed

---

## Goal

Correctly model the two-phase transition: create the run as `pending` at request time, then transition it to `running` (with `started_at`) inside the goroutine when execution actually begins.

---

## Acceptance Criteria

- [x] `CreateWorkflowRun` inserts the row with `status = "pending"`
- [x] The goroutine sets `status = "running"` and `started_at = NOW()` before calling `engine.Execute`
- [x] `completed_at` is set correctly when the run finishes (success or failure)
- [x] `started_at` is never NULL on a completed or failed run
- [x] All existing router tests pass
- [x] New tests cover the lifecycle transitions

> **Status: COMPLETE** — `CreateWorkflowRun` now uses `"pending"` status. Added `transitionToRunning()` method that updates status to `"running"` with `started_at` timestamp inside the goroutine before engine execution. Test: `TestTriggerHandler_CreatesRunAsPending`.

---

## TDD Approach

### Red — write failing tests first

**File:** `internal/api/router_test.go`

```go
// TestWorkflowTrigger_RunCreatedAsPending verifies that immediately after
// the trigger endpoint returns, the run exists in the DB with status "pending".
// (Requires a test DB or mock Queries.)
func TestWorkflowTrigger_RunCreatedAsPending(t *testing.T) { ... }
```

Since the current tests use `httptest` without a real DB, the most practical approach is to mock `Queries` or use a test DB (see the existing pattern in `router_test.go`). Focus on the HTTP response contract first (202 Accepted, `run_id` present) and add a DB assertion if the test setup supports it.

### Green — minimal implementation

1. Change the `CreateWorkflowRun` call to use `status = "pending"`:
   ```go
   _, err = a.queries.CreateWorkflowRun(c, db.CreateWorkflowRunParams{
       ID:         pgRunID,
       WorkflowID: pgID,
       Status:     "pending",
       InputData:  inputBytes,
   })
   ```

2. At the start of the goroutine, before calling `engine.Execute`, update the run to `running`:
   ```go
   go func(runID pgtype.UUID, w db.Workflow, initialData map[string]any) {
       ctx := context.Background()
       now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
       a.queries.UpdateWorkflowRunStatus(ctx, db.UpdateWorkflowRunStatusParams{
           ID:        runID,
           Status:    "running",
           StartedAt: now,  // set started_at here
       })
       // ... rest of execution
   }
   ```

3. Confirm that `UpdateWorkflowRunStatus` in `query.sql` already supports `started_at` via `sqlc.narg` — it does (line 35 of `query.sql`). No SQL change is needed.

### Refactor

- The `updateRunStatus` helper should accept a `startedAt *time.Time` parameter so it can set `started_at` in one call when transitioning to `running`.

---

## Definition of Done

- `go test ./...` passes
- A run created by the trigger endpoint has `status = "pending"` in the DB immediately
- After the goroutine runs, `started_at` is non-NULL
- `completed_at` is set on every terminal run (completed or failed)
