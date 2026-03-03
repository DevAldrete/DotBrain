# HTTP API

**Source:** `internal/api/router.go`

All endpoints are under the `/api/v1` prefix. The server uses Gin with `Recovery` and `Logger` middleware applied globally.

There is no authentication. All endpoints are publicly accessible.

---

## Infrastructure

### `GET /api/v1/health`

Liveness probe. Returns 200 if the process is running. Kubernetes uses this to determine whether to restart a pod.

**Response 200:**
```json
{
  "status": "UP",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

---

### `GET /api/v1/readiness`

Readiness probe. Pings the database connection pool. Returns 503 if the DB is unreachable, signaling Kubernetes not to route traffic to this pod.

**Response 200:**
```json
{
  "status": "READY",
  "message": "Service is ready to accept traffic"
}
```

**Response 503:**
```json
{
  "status": "NOT_READY",
  "message": "Database connection failed",
  "error": "..."
}
```

---

## Workflows

### `POST /api/v1/workflows`

Creates a new workflow definition and persists it to the database.

**Request body:**
```json
{
  "name": "My Pipeline",
  "description": "Optional description",
  "definition": {
    "nodes": [
      { "id": "step-1", "type": "echo" }
    ]
  }
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | |
| `description` | string | no | Defaults to `""` |
| `definition` | object | yes | Must be a valid `WorkflowDefinition` shape. Node types are **not** validated at creation time. |

**Response 201** — the created workflow row:
```json
{
  "ID": "019547a2-...",
  "Name": "My Pipeline",
  "Description": "Optional description",
  "Definition": "<raw JSONB bytes as base64 or string>",
  "CreatedAt": "2025-01-15T10:30:00Z",
  "UpdatedAt": "2025-01-15T10:30:00Z"
}
```

> The `Definition` field is returned as raw bytes from the sqlc-generated struct. Clients should `JSON.parse` it if needed.

**Response 400:** invalid JSON body or missing required fields.
**Response 500:** database error.

---

### `GET /api/v1/workflows`

Lists all workflows, ordered by `created_at DESC`. Returns up to 100 results (hardcoded; no pagination parameters).

**Response 200:**
```json
[
  {
    "ID": "019547a2-...",
    "Name": "My Pipeline",
    ...
  }
]
```

Returns `[]` (empty array) when no workflows exist.

**Response 500:** database error.

---

### `GET /api/v1/workflows/:id`

Retrieves a single workflow by UUID.

**Path param:** `id` — UUID v7 string.

**Response 200:** single workflow object (same shape as the list response).

**Response 400:** `id` is not a valid UUID.
**Response 404:** no workflow with that ID.

---

### `POST /api/v1/workflows/:id/trigger`

Triggers a new execution of the workflow. Creates a `workflow_run` row and immediately returns a `run_id`. Execution happens asynchronously in a goroutine.

**Path param:** `id` — workflow UUID.

**Request body:** arbitrary JSON object. This becomes `input_data` for the run and the initial `input` map passed to the first node.

```json
{ "article_id": "42", "user": "alice" }
```

An empty object `{}` is valid.

**Response 202:**
```json
{
  "message": "workflow queued for execution",
  "run_id": "019547b3-..."
}
```

Use the `run_id` to poll `GET /api/v1/runs/:id` for status.

**Response 400:** `id` is not a valid UUID, or request body is not valid JSON.
**Response 404:** workflow not found.
**Response 500:** failed to create the `workflow_run` row.

> The response is 202 even if the workflow definition is invalid. Definition parsing and node loading happen inside the goroutine *after* the response is sent. An invalid definition will result in the run being marked `failed` asynchronously.

---

### `GET /api/v1/workflows/:id/runs`

Lists all runs for a workflow, ordered by `created_at DESC`. Returns up to 100 results.

**Path param:** `id` — workflow UUID.

**Response 200:**
```json
[
  {
    "ID": "019547b3-...",
    "WorkflowID": "019547a2-...",
    "Status": "completed",
    "InputData": "...",
    "OutputData": "...",
    "Error": null,
    "StartedAt": "2025-01-15T10:30:01Z",
    "CompletedAt": "2025-01-15T10:30:03Z",
    "CreatedAt": "2025-01-15T10:30:00Z"
  }
]
```

Returns `[]` when no runs exist.

**Response 400:** invalid workflow UUID.
**Response 500:** database error.

---

## Runs

### `GET /api/v1/runs/:id`

Retrieves a single workflow run by UUID.

**Path param:** `id` — run UUID.

**Response 200:** single run object (same shape as the list response).

**Response 400:** `id` is not a valid UUID.
**Response 404:** run not found.

---

### `GET /api/v1/runs/:id/nodes`

Lists all node execution records for a run, ordered by `created_at ASC` (execution order).

**Path param:** `id` — run UUID.

**Response 200:**
```json
[
  {
    "ID": "019547c1-...",
    "WorkflowRunID": "019547b3-...",
    "NodeID": "fetch-content",
    "Status": "completed",
    "InputData": "...",
    "OutputData": "...",
    "Error": null,
    "StartedAt": "2025-01-15T10:30:01Z",
    "CompletedAt": "2025-01-15T10:30:02Z",
    "CreatedAt": "2025-01-15T10:30:01Z"
  },
  {
    "ID": "019547c2-...",
    "NodeID": "summarize",
    "Status": "completed",
    ...
  }
]
```

Returns `[]` when no node executions exist (e.g., the run failed before any node started).

**Response 400:** invalid run UUID.
**Response 500:** database error.

---

## Error Response Shape

All error responses use the same envelope:

```json
{ "error": "human-readable message" }
```

---

## Endpoint Summary

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/health` | Liveness probe |
| `GET` | `/api/v1/readiness` | Readiness probe (DB ping) |
| `POST` | `/api/v1/workflows` | Create workflow |
| `GET` | `/api/v1/workflows` | List workflows |
| `GET` | `/api/v1/workflows/:id` | Get workflow |
| `POST` | `/api/v1/workflows/:id/trigger` | Trigger a run |
| `GET` | `/api/v1/workflows/:id/runs` | List runs for a workflow |
| `GET` | `/api/v1/runs/:id` | Get run status and output |
| `GET` | `/api/v1/runs/:id/nodes` | Get per-node execution detail |

**Not yet implemented:** `PUT /workflows/:id` (update), `DELETE /workflows/:id` (delete), `POST /runs/:id/cancel` (cancel).

---

## Frontend API Client

**Source:** `web/src/lib/api.ts`

The SvelteKit frontend wraps all endpoints in typed async functions. All functions throw `ApiError` (with a `.status` HTTP code) on non-2xx responses.

```ts
import {
  listWorkflows,       // GET /workflows
  getWorkflow,         // GET /workflows/:id
  createWorkflow,      // POST /workflows
  triggerWorkflow,     // POST /workflows/:id/trigger
  listWorkflowRuns,    // GET /workflows/:id/runs
  getWorkflowRun,      // GET /runs/:id
  listNodeExecutions,  // GET /runs/:id/nodes
  ApiError
} from '$lib/api';
```

The base URL is `/api/v1`, which is proxied to the Go server via the Vite dev server config.
