# HTTP API

**Source:** `internal/api/router.go`

All endpoints are under the `/api/v1` prefix. The server uses Gin with `Recovery` and `Logger` middleware applied globally.

## Authentication

All `/api/v1` routes **except `/health` and `/readiness`** require an `Authorization` header with a valid API key:

```
Authorization: Bearer <your-api-key>
```

The API key is configured via the `API_KEY` environment variable on the server. When `API_KEY` is empty (local development), authentication is disabled and all routes are accessible without a header. A startup log line indicates whether auth is enabled or disabled.

Requests missing the header, using a malformed header, or supplying an incorrect key receive **401 Unauthorized**:

```json
{ "error": "missing Authorization header" }
{ "error": "Authorization header must be in the format: Bearer <token>" }
{ "error": "invalid API key" }
```

The frontend (`web/src/lib/api.ts`) reads the key from `VITE_API_KEY` at build time and includes it in every request automatically.

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
    "nodes": [{ "id": "step-1", "type": "echo" }]
  }
}
```

| Field         | Type   | Required | Notes                                                                                          |
| ------------- | ------ | -------- | ---------------------------------------------------------------------------------------------- |
| `name`        | string | yes      |                                                                                                |
| `description` | string | no       | Defaults to `""`                                                                               |
| `definition`  | object | yes      | Must be a valid `WorkflowDefinition` shape. Node types are **not** validated at creation time. |

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

### `PUT /api/v1/workflows/:id`

Replaces all mutable fields of a workflow (full replacement, not partial patch). Uses the same request body shape as `POST /api/v1/workflows`.

**Path param:** `id` — UUID v7 string.

**Request body:**

```json
{
  "name": "Updated Pipeline Name",
  "description": "Updated description",
  "definition": {
    "nodes": [{ "id": "step-1", "type": "echo" }]
  }
}
```

All fields are required. `updated_at` is set to `NOW()`. The `id` and `created_at` are never modified.

> Existing runs keep the definition they were triggered with. Updating a workflow does not retroactively change past runs.

**Response 200:** the updated workflow object.
**Response 400:** invalid UUID, invalid JSON, or missing required fields.
**Response 404:** workflow not found.
**Response 500:** database error.

---

### `DELETE /api/v1/workflows/:id`

Permanently deletes the workflow and all its associated runs and node executions (via `ON DELETE CASCADE`).

**Path param:** `id` — UUID v7 string.

**Response 204:** no content.
**Response 400:** invalid UUID.
**Response 404:** workflow not found.
**Response 500:** database error.

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

> The response is 202 even if the workflow definition is invalid. Definition parsing and node loading happen inside the goroutine _after_ the response is sent. An invalid definition will result in the run being marked `failed` asynchronously.

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

### `GET /api/v1/runs/:id/stream`

Streams real-time Server-Sent Events (SSE) as a run progresses.

**Path param:** `id` — run UUID.

**Response headers:**

```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

**Response 400:** `id` is not a valid UUID.
**Response 404:** run not found.

#### Behaviour

- If the run is already in a terminal state (`completed`, `failed`, `cancelled`) when the client connects, the server emits the corresponding terminal event immediately and closes the stream.
- For an active run, the server streams events as they are published by the workflow engine.
- After a terminal event the server closes the stream. The browser `EventSource` API will **not** auto-reconnect once the server closes the connection.
- If the client disconnects (tab closed, navigation away), the server detects the context cancellation and stops sending without leaking goroutines.
- Multiple concurrent clients may subscribe to the same run — each receives all events independently.

#### Event Catalog

Each SSE message has the form:

```
event: <type>
data: <JSON payload>

```

| Event type       | Emitted when                             | Payload fields                     |
| ---------------- | ---------------------------------------- | ---------------------------------- |
| `run.started`    | Run begins executing                     | `run_id`, `status`                 |
| `node.started`   | A node begins executing                  | `node_id`, `input`                 |
| `node.completed` | A node finishes successfully             | `node_id`, `output`, `duration_ms` |
| `node.failed`    | A node fails permanently                 | `node_id`, `error`                 |
| `node.retrying`  | A node is about to be retried            | `node_id`, `attempt`, `error`      |
| `run.completed`  | Run finishes successfully (**terminal**) | `run_id`, `status`, `output`       |
| `run.failed`     | Run fails (**terminal**)                 | `run_id`, `status`, `error`        |
| `run.cancelled`  | Run is cancelled (**terminal**)          | `run_id`, `status`                 |

#### Example stream

```
event: run.started
data: {"run_id":"019547b3-...","status":"running"}

event: node.started
data: {"node_id":"fetch-content","input":{"url":"https://example.com"}}

event: node.completed
data: {"node_id":"fetch-content","output":{"body":"..."},"duration_ms":342}

event: run.completed
data: {"run_id":"019547b3-...","status":"completed","output":{"body":"..."}}
```

#### Frontend usage

The run detail page (`web/src/routes/runs/[id]/+page.svelte`) opens an `EventSource` automatically when the run is active and shows a **LIVE** indicator:

```ts
const source = new EventSource(`/api/v1/runs/${runId}/stream`);

source.addEventListener("node.completed", (e) => {
  const data = JSON.parse(e.data);
  // data.node_id, data.output, data.duration_ms
});

source.addEventListener("run.completed", (e) => {
  source.close();
});
```

---

## Error Response Shape

All error responses use the same envelope:

```json
{ "error": "human-readable message" }
```

---

## Endpoint Summary

| Method   | Path                              | Description                      |
| -------- | --------------------------------- | -------------------------------- |
| `GET`    | `/api/v1/health`                  | Liveness probe                   |
| `GET`    | `/api/v1/readiness`               | Readiness probe (DB ping)        |
| `POST`   | `/api/v1/workflows`               | Create workflow                  |
| `GET`    | `/api/v1/workflows`               | List workflows                   |
| `GET`    | `/api/v1/workflows/:id`           | Get workflow                     |
| `PUT`    | `/api/v1/workflows/:id`           | Update workflow                  |
| `DELETE` | `/api/v1/workflows/:id`           | Delete workflow (cascades runs)  |
| `POST`   | `/api/v1/workflows/:id/trigger`   | Trigger a run                    |
| `GET`    | `/api/v1/workflows/:id/runs`      | List runs for a workflow         |
| `GET`    | `/api/v1/runs/:id`                | Get run status and output        |
| `GET`    | `/api/v1/runs/:id/nodes`          | Get per-node execution detail    |
| `GET`    | `/api/v1/runs/:id/stream`         | Stream live SSE events for a run |
| `POST`   | `/api/v1/runs/:id/cancel`         | Cancel an in-progress run        |
| `POST`   | `/api/v1/workflows/:id/schedules` | Create a cron schedule           |
| `GET`    | `/api/v1/workflows/:id/schedules` | List schedules for a workflow    |
| `DELETE` | `/api/v1/schedules/:id`           | Delete a schedule                |
| `PATCH`  | `/api/v1/schedules/:id`           | Update a schedule                |

---

## Frontend API Client

**Source:** `web/src/lib/api.ts`

The SvelteKit frontend wraps all endpoints in typed async functions. All functions throw `ApiError` (with a `.status` HTTP code) on non-2xx responses.

```ts
import {
  listWorkflows, // GET /workflows
  getWorkflow, // GET /workflows/:id
  createWorkflow, // POST /workflows
  updateWorkflow, // PUT /workflows/:id
  deleteWorkflow, // DELETE /workflows/:id
  triggerWorkflow, // POST /workflows/:id/trigger
  listWorkflowRuns, // GET /workflows/:id/runs
  getWorkflowRun, // GET /runs/:id
  listNodeExecutions, // GET /runs/:id/nodes
  cancelRun, // POST /runs/:id/cancel
  createSchedule, // POST /workflows/:id/schedules
  listSchedules, // GET /workflows/:id/schedules
  deleteSchedule, // DELETE /schedules/:id
  updateSchedule, // PATCH /schedules/:id
  ApiError,
} from "$lib/api";
```

The base URL is `/api/v1`, proxied to the Go server via the Vite dev server config. The `VITE_API_KEY` env var is included automatically in every request header.
