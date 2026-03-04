# Data Model

**Source:** `schema.sql`, `query.sql`, `internal/db/sqlc/`

DotBrain uses three PostgreSQL tables. The schema is the source of truth; Go structs are generated from it via sqlc.

---

## Tables

### `workflows`

Stores the static definition of a workflow. A workflow is never mutated after creation (no update endpoint exists yet).

```sql
CREATE TABLE workflows (
    id          UUID PRIMARY KEY,           -- UUID v7
    name        VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    definition  JSONB NOT NULL,             -- WorkflowDefinition JSON
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

The `definition` column stores the full `WorkflowDefinition` JSON blob. It is not indexed or queryable by content — the entire document is loaded at trigger time.

---

### `workflow_runs`

One row per execution instance. Created when a trigger is received; updated as the run progresses to completion or failure.

```sql
CREATE TABLE workflow_runs (
    id           UUID PRIMARY KEY,
    workflow_id  UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status       VARCHAR(50) NOT NULL DEFAULT 'pending',
    input_data   JSONB,          -- trigger payload
    output_data  JSONB,          -- last node's output on success
    error        TEXT,           -- error message on failure
    started_at   TIMESTAMP WITH TIME ZONE,   -- set when goroutine begins
    completed_at TIMESTAMP WITH TIME ZONE,   -- set when run ends
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflow_runs_status ON workflow_runs(status);
```

**Run status values:**

| Status | Meaning |
|---|---|
| `pending` | Run row created; goroutine not yet started |
| `running` | Goroutine is active; nodes are executing |
| `completed` | All nodes finished successfully |
| `failed` | A node returned an error, definition parsing failed, or crash recovery marked the run as failed |
| `cancelled` | Reserved; not yet implemented |

---

### `node_executions`

One row per node per run. Provides a per-step audit trail.

```sql
CREATE TABLE node_executions (
    id               UUID PRIMARY KEY,
    workflow_run_id  UUID NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    node_id          VARCHAR(255) NOT NULL,   -- NodeConfig.ID from the definition
    status           VARCHAR(50) NOT NULL DEFAULT 'pending',
    input_data       JSONB,
    output_data      JSONB,
    error            TEXT,
    started_at       TIMESTAMP WITH TIME ZONE,
    completed_at     TIMESTAMP WITH TIME ZONE,
    created_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(workflow_run_id, node_id)
);

CREATE INDEX idx_node_executions_status ON node_executions(status);
CREATE INDEX idx_node_executions_run_id ON node_executions(workflow_run_id);
```

The `UNIQUE(workflow_run_id, node_id)` constraint enforces idempotency: each node ID can only appear once per run. This means if the same `node_id` appears twice in a workflow definition, the second `OnNodeStart` call will fail silently (the insert is ignored via `_`-discarded error in `DBNodeHook`).

**Node execution status values:**

| Status | Meaning |
|---|---|
| `pending` | Row not yet created (used conceptually; rows are inserted as `running`) |
| `running` | `INSERT` written by `OnNodeStart`; node is executing |
| `completed` | `UPDATE` written by `OnNodeComplete` |
| `failed` | `UPDATE` written by `OnNodeFail` |
| `retrying` | Reserved; not yet implemented |

---

## Run Lifecycle State Machine

```
                  POST /workflows/:id/trigger
                            │
                            ▼
                   ┌─────────────────┐
                   │    pending      │  ← workflow_run INSERT
                   └────────┬────────┘
                            │  goroutine starts
                            ▼
                   ┌─────────────────┐
                   │    running      │  ← transitionToRunning()
                   └────────┬────────┘
                            │
               ┌────────────┴────────────┐
               │                         │
               ▼                         ▼
    ┌─────────────────┐       ┌─────────────────┐
    │   completed     │       │     failed      │
    └─────────────────┘       └─────────────────┘
                                       ▲
                                       │
                              crash recovery /
                              watchdog timeout
                              (from pending or
                               running)
```

`completed` and `failed` are terminal states. Once set, no further updates occur.

**What can cause `failed`:**
- `ParseDefinition` returns an error (malformed JSON)
- `LoadFromDefinition` returns an error (unknown node type)
- Any `NodeExecutor.Execute` returns an error
- **Crash recovery at startup:** `RecoverStaleRuns` marks any runs stuck in `running` or `pending` as `failed` with the error `"run aborted: server restarted while execution was in progress"`
- **Watchdog timeout:** `FailTimedOutRuns` marks `running` runs whose `started_at` exceeds `RUN_MAX_DURATION` (default 1h) as `failed` with the error `"run timed out: exceeded maximum duration of <duration>"`

### Crash Recovery

On startup, `RecoverStaleRuns` is called immediately after `NewAPI` to fail any runs left in a non-terminal state from a previous crash. This prevents runs from being stuck in `running` or `pending` indefinitely.

A background watchdog goroutine (`RunWatchdog`) runs on a configurable interval (`WATCHDOG_INTERVAL`, default 5m) and calls `FailTimedOutRuns` to mark runs that have exceeded `RUN_MAX_DURATION` (default 1h) as failed. The watchdog is cancelled during graceful shutdown.

**Environment variables:**

| Variable | Default | Description |
|---|---|---|
| `RUN_MAX_DURATION` | `1h` | Maximum allowed duration for a running workflow run |
| `WATCHDOG_INTERVAL` | `5m` | How often the watchdog checks for timed-out runs |

---

## Generated Go Types (sqlc)

**Source:** `internal/db/sqlc/models.go`

```go
type Workflow struct {
    ID          pgtype.UUID
    Name        string
    Description string
    Definition  []byte        // raw JSONB bytes; parse with core.ParseDefinition
    CreatedAt   pgtype.Timestamptz
    UpdatedAt   pgtype.Timestamptz
}

type WorkflowRun struct {
    ID          pgtype.UUID
    WorkflowID  pgtype.UUID
    Status      string
    InputData   []byte
    OutputData  []byte
    Error       pgtype.Text
    StartedAt   pgtype.Timestamptz
    CompletedAt pgtype.Timestamptz
    CreatedAt   pgtype.Timestamptz
}

type NodeExecution struct {
    ID             pgtype.UUID
    WorkflowRunID  pgtype.UUID
    NodeID         string
    Status         string
    InputData      []byte
    OutputData     []byte
    Error          pgtype.Text
    StartedAt      pgtype.Timestamptz
    CompletedAt    pgtype.Timestamptz
    CreatedAt      pgtype.Timestamptz
}
```

`pgtype.Text` fields (like `Error`) use `.Valid` to distinguish SQL NULL from an empty string. `[]byte` JSONB fields are raw JSON; callers must `json.Unmarshal` them.

---

## Available Queries

**Source:** `query.sql` → generated into `internal/db/sqlc/query.sql.go`

| Query name | Operation | Used by |
|---|---|---|
| `CreateWorkflow` | INSERT | `createWorkflowHandler` |
| `GetWorkflow` | SELECT by ID | `getWorkflowHandler`, `workflowTriggerHandler`, `deleteWorkflowHandler` |
| `UpdateWorkflow` | UPDATE name/description/definition, RETURNING * | `updateWorkflowHandler` |
| `DeleteWorkflow` | DELETE by ID | `deleteWorkflowHandler` |
| `ListWorkflows` | SELECT all (paginated) | `listWorkflowsHandler` |
| `CreateWorkflowRun` | INSERT | `workflowTriggerHandler` |
| `GetWorkflowRun` | SELECT by ID | `getRunHandler` |
| `UpdateWorkflowRunStatus` | UPDATE (partial, via COALESCE) | `transitionToRunning`, `updateRunStatus` |
| `ListWorkflowRuns` | SELECT by workflow_id (paginated) | `listWorkflowRunsHandler` |
| `CreateNodeExecution` | INSERT | `DBNodeHook.OnNodeStart` |
| `GetNodeExecution` | SELECT by ID | (unused in current handlers) |
| `UpdateNodeExecutionStatus` | UPDATE (partial, via COALESCE) | `DBNodeHook.OnNodeComplete`, `DBNodeHook.OnNodeFail` |
| `ListNodeExecutionsForRun` | SELECT by run_id (ordered by created_at) | `listNodeExecutionsHandler` |
| `ListPendingNodeExecutions` | SELECT where status='pending' | (defined but unused) |
| `FailStaleRuns` | UPDATE running/pending → failed | `RecoverStaleRuns` (startup) |
| `FailRunsExceedingDuration` | UPDATE running past threshold → failed | `FailTimedOutRuns` (watchdog) |

`UpdateWorkflowRunStatus` and `UpdateNodeExecutionStatus` use `COALESCE(sqlc.narg(...), existing_value)` — this means passing a nil/zero value for an optional field leaves the existing column value unchanged, enabling partial updates without a full struct.

---

## Cascade Deletes

Both `workflow_runs` and `node_executions` have `ON DELETE CASCADE` foreign keys:

- Deleting a `workflow` row deletes all its `workflow_runs`.
- Deleting a `workflow_run` row deletes all its `node_executions`.

There is no delete endpoint in the current API, so this only matters for manual DB operations.
