# TASK-03 — Add Run Queries and API Endpoints

**Phase:** 2 — Expand the API Surface  
**Priority:** High (required by TASK-07 and TASK-08)  
**Depends on:** TASK-04 (run lifecycle) and TASK-02 (node executions) for meaningful data  
**Files affected:** `query.sql`, `internal/db/sqlc/query.sql.go` (regenerated), `internal/api/router.go`

---

## Problem

The dashboard needs to display run history and node-level detail, but three required queries are missing from `query.sql` and three API endpoints are missing from the router:

**Missing queries:**
- `ListWorkflowRuns` — list all runs for a given workflow ID, ordered by creation time
- `ListNodeExecutionsForRun` — list all node execution records for a given run ID
- `UpdateWorkflow` — update a workflow's name, description, or definition (needed for editing)

**Missing endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/workflows/:id/runs` | List runs for a workflow (paginated) |
| `GET` | `/api/v1/runs/:id` | Get a single run by ID |
| `GET` | `/api/v1/runs/:id/nodes` | List node executions for a run |

> `GetWorkflowRun :one` already exists in `query.sql` (line 26), so `GET /api/v1/runs/:id` only needs a handler, not a new query.

---

## Goal

Add the missing SQL queries, regenerate the sqlc code, and wire up the three new HTTP endpoints.

---

## Acceptance Criteria

- [ ] `query.sql` contains `ListWorkflowRuns`, `ListNodeExecutionsForRun`
- [ ] `sqlc generate` runs without errors and produces updated `query.sql.go`
- [ ] `GET /api/v1/workflows/:id/runs` returns a JSON array of `WorkflowRun` objects, ordered by `created_at DESC`; supports optional `?limit=` and `?offset=` query params
- [ ] `GET /api/v1/runs/:id` returns a single `WorkflowRun` or 404
- [ ] `GET /api/v1/runs/:id/nodes` returns a JSON array of `NodeExecution` objects for the run, ordered by `created_at ASC`
- [ ] All three endpoints return `[]` (not `null`) when no results are found
- [ ] All three endpoints are covered by handler tests in `router_test.go`

---

## SQL to Add to `query.sql`

```sql
-- name: ListWorkflowRuns :many
SELECT * FROM workflow_runs
WHERE workflow_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListNodeExecutionsForRun :many
SELECT * FROM node_executions
WHERE workflow_run_id = $1
ORDER BY created_at ASC;
```

After adding these, run:
```bash
sqlc generate
```

---

## TDD Approach

### Red — write failing tests first

**File:** `internal/api/router_test.go`

```go
// TestListWorkflowRuns_ReturnsEmptyArray verifies the endpoint returns []
// (not null or 404) when a workflow has no runs.
func TestListWorkflowRuns_ReturnsEmptyArray(t *testing.T) {
    // POST a workflow, then GET /api/v1/workflows/:id/runs
    // Expect 200 and body = []
}

// TestGetWorkflowRun_NotFound verifies 404 for an unknown run ID.
func TestGetWorkflowRun_NotFound(t *testing.T) {
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/runs/"+uuid.New().String(), nil)
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestListNodeExecutions_ReturnsEmptyArray verifies the endpoint returns []
// when a run has no node execution records.
func TestListNodeExecutions_ReturnsEmptyArray(t *testing.T) { ... }
```

### Green — minimal implementation

**Router changes** (`internal/api/router.go`):

1. Register new routes in `NewRouter`:
   ```go
   v1.GET("/workflows/:id/runs", a.listWorkflowRunsHandler)
   v1.GET("/runs/:id", a.getWorkflowRunHandler)
   v1.GET("/runs/:id/nodes", a.listNodeExecutionsHandler)
   ```

2. Implement `listWorkflowRunsHandler`:
   - Parse `:id`, optional `limit` (default 50) and `offset` (default 0) query params
   - Call `a.queries.ListWorkflowRuns(c, db.ListWorkflowRunsParams{WorkflowID: pgID, Limit: limit, Offset: offset})`
   - Return 200 with array (empty array if nil)

3. Implement `getWorkflowRunHandler`:
   - Parse `:id`
   - Call `a.queries.GetWorkflowRun(c, pgID)`
   - Return 200 or 404

4. Implement `listNodeExecutionsHandler`:
   - Parse run `:id`
   - Call `a.queries.ListNodeExecutionsForRun(c, pgID)`
   - Return 200 with array (empty array if nil)

### Refactor

- Extract the repeated UUID parse + pgtype.UUID construction into a helper `parseUUID(idStr string) (pgtype.UUID, error)` to reduce boilerplate across all handlers.

---

## Definition of Done

- `sqlc generate` succeeds
- `go test ./...` passes with new handler tests
- All three endpoints return correct data when tested with `curl` against a running instance
- The router registration in `NewRouter` is documented with a comment grouping run-related endpoints together
