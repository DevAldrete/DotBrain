# DotBrain — Architecture Overview

## What Is DotBrain?

DotBrain is a workflow orchestration engine. You define a workflow as an ordered list of nodes (steps), store it in the database, and then trigger it with an input payload. The engine executes each node sequentially, piping the output of one node as the input to the next, and records the result.

The intended use cases are:

- **LLM/AI pipelines** — chains of OpenAI (or Anthropic) calls, prompt transformations, and validation steps
- **General automation** — HTTP requests, data transformation, schema validation, conditional branching

---

## Repository Structure

```
DotBrain/
├── cmd/dotbrain/main.go          # Application entrypoint — reads config, starts HTTP server
├── internal/
│   ├── api/
│   │   ├── router.go             # Gin HTTP handlers and route registration
│   │   └── router_test.go        # Integration-style tests using httptest
│   └── core/
│       ├── workflow.go           # WorkflowDefinition and NodeConfig data structures
│       ├── engine.go             # Engine — sequential node orchestrator + node registry
│       ├── node.go               # NodeExecutor interface; EchoNode, FailNode, MathNode
│       ├── llm_node.go           # LLMNode (stub — not yet registered or wired to OpenAI)
│       └── safe_object_node.go   # SafeObjectNode — schema-validating node (not yet registered)
├── internal/db/sqlc/
│   ├── db.go                     # sqlc DBTX interface and Queries struct
│   ├── models.go                 # Generated Go structs: Workflow, WorkflowRun, NodeExecution
│   └── query.sql.go              # Generated typed query methods
├── web/                          # SvelteKit frontend (currently a static landing page)
├── schema.sql                    # PostgreSQL DDL — source of truth for the database schema
├── query.sql                     # sqlc query definitions — source of truth for DB queries
├── sqlc.yaml                     # sqlc code-generation config
├── docker-compose.yml            # API + PostgreSQL (port 5432) services
├── Dockerfile                    # Multi-stage build: golang:1.25-alpine → alpine:3.19
└── Justfile                      # Developer task runner (default, format, test)
```

---

## Technology Stack

| Layer | Technology | Notes |
|-------|-----------|-------|
| Language | Go 1.25 | Backend |
| HTTP | Gin v1.11 | Request routing and middleware |
| Database | PostgreSQL 17 | Hosted via Docker in dev |
| DB access | sqlc + pgx/v5 | Type-safe generated queries; no ORM |
| IDs | UUID v7 | Monotonic, index-friendly |
| Logging | `log/slog` | Structured JSON output |
| Frontend | SvelteKit 2 + Svelte 5 | TypeScript, Tailwind CSS v4 |
| Dev reload | Air | Watches `*.go` files, restarts binary |
| Containers | Docker Compose | `api` and `db` services |

---

## Data Model

### `workflows`

Stores the structural definition of a workflow. The `definition` column is a JSONB document mapping to `core.WorkflowDefinition`.

```
id           UUID (PK, v7)
name         VARCHAR(255)
description  TEXT
definition   JSONB   -- {"nodes": [{"id": "...", "type": "...", "params": {...}}]}
created_at   TIMESTAMPTZ
updated_at   TIMESTAMPTZ
```

### `workflow_runs`

An instance of a workflow being executed. Created when a trigger is received; updated as the run progresses.

```
id           UUID (PK, v7)
workflow_id  UUID (FK → workflows.id, CASCADE DELETE)
status       VARCHAR(50)  -- pending | running | completed | failed | cancelled
input_data   JSONB        -- the payload sent at trigger time
output_data  JSONB        -- the final node's output on success
error        TEXT         -- error message if status = failed
started_at   TIMESTAMPTZ  -- set when execution begins
completed_at TIMESTAMPTZ  -- set when execution ends
created_at   TIMESTAMPTZ
```

### `node_executions`

Per-step audit record within a run. One row per node per run.

```
id               UUID (PK, v7)
workflow_run_id  UUID (FK → workflow_runs.id, CASCADE DELETE)
node_id          VARCHAR(255)  -- matches NodeConfig.ID from the definition
status           VARCHAR(50)   -- pending | running | completed | failed | retrying
input_data       JSONB
output_data      JSONB
error            TEXT
started_at       TIMESTAMPTZ
completed_at     TIMESTAMPTZ
created_at       TIMESTAMPTZ
UNIQUE(workflow_run_id, node_id)
```

---

## Core Execution Model

### `WorkflowDefinition` and `NodeConfig`

A workflow is defined as a JSON document:

```json
{
  "nodes": [
    { "id": "step-1", "type": "echo" },
    { "id": "step-2", "type": "math", "params": { "a": 1, "b": 2 } },
    { "id": "step-3", "type": "llm",  "params": { "prompt": "Summarize: {{input.result}}" } }
  ]
}
```

- `id` — unique identifier for this step within the workflow (used for `node_executions` records)
- `type` — must match a key in `engine.nodeRegistry`
- `params` — arbitrary key-value map passed to the node at instantiation time

### `NodeExecutor` Interface

Every node implements:

```go
type NodeExecutor interface {
    Execute(ctx context.Context, input map[string]any) (map[string]any, error)
}
```

The `input` map for the first node is the trigger payload. Each subsequent node receives the previous node's output map as its input.

### `Engine`

The `Engine` holds a `[]NodeExecutor` slice. `Execute` iterates sequentially; if any node returns an error, execution stops and the run is marked `failed`.

The **node registry** maps type name → factory function:

```go
var nodeRegistry = map[string]func(params map[string]any) NodeExecutor{
    "echo": func(p map[string]any) NodeExecutor { return EchoNode{} },
    "math": func(p map[string]any) NodeExecutor { return MathNode{Params: p} },
    "http": func(p map[string]any) NodeExecutor { return HttpNode{Params: p} },
    "llm":  func(p map[string]any) NodeExecutor { return LLMNode{Params: p} },
}
```

> Note: The registry signature above reflects the target state after TASK-01.

---

## API Endpoints

All routes are under `/api/v1`:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/health` | Liveness probe — always 200 if the process is running |
| `GET` | `/api/v1/readiness` | Readiness probe — pings the DB; returns 503 if unavailable |
| `GET` | `/api/v1/ping` | Smoke test |
| `POST` | `/api/v1/workflows` | Create a workflow definition |
| `GET` | `/api/v1/workflows` | List all workflows |
| `GET` | `/api/v1/workflows/:id` | Get a single workflow |
| `POST` | `/api/v1/workflows/:id/trigger` | Trigger a workflow run |
| `GET` | `/api/v1/workflows/:id/runs` | List runs for a workflow _(planned — TASK-03)_ |
| `GET` | `/api/v1/runs/:id` | Get a single run _(planned — TASK-03)_ |
| `GET` | `/api/v1/runs/:id/nodes` | Get node executions for a run _(planned — TASK-03)_ |

---

## Workflow Run Lifecycle

```
trigger received
      │
      ▼
CREATE workflow_run (status = "pending")
      │
      ▼
  goroutine spawned → started_at = NOW(), status = "running"
      │
      ▼
  for each node:
    CREATE node_execution (status = "running")
    node.Execute(ctx, input)
    ├─ success → UPDATE node_execution (status = "completed")
    └─ failure → UPDATE node_execution (status = "failed")
                 UPDATE workflow_run (status = "failed")
                 RETURN
      │
      ▼
  last node completed
      │
      ▼
UPDATE workflow_run (status = "completed", output_data = last output, completed_at = NOW())
```

> Note: The node_execution records and the `pending → running` status transition are the target state after TASK-02 and TASK-04.

---

## Configuration

Environment variables (see `.env.example`):

| Variable | Required | Description |
|----------|----------|-------------|
| `PORT` | Yes | HTTP listen port (default `8080`) |
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `ENV` | No | `development` or `production` |
| `OPENAI_API_KEY` | For LLM nodes | OpenAI API key |
| `ANTHROPIC_API_KEY` | Future | Anthropic API key |
| `SECRET_KEY` | Future | JWT / HMAC signing key |

---

## Development Workflow

```bash
# Start the DB and API with live reload
docker compose up db -d
just default             # runs Air for hot-reload

# Run all tests
just test                # go test ./...

# Regenerate DB query code after editing query.sql
sqlc generate

# Build the frontend
cd web && npm install && npm run build
```

---

## Node Catalog

| Type | Status | Description |
|------|--------|-------------|
| `echo` | Stable | Passes input through unchanged. Useful for debugging. |
| `fail` | Stable | Always fails. Used in tests. |
| `math` | Stable | Adds `input["a"]` + `input["b"]`, returns `{"result": sum}`. |
| `safe_object` | Implemented, not registered | Validates and filters input against a type schema. |
| `http` | Planned (TASK-05) | Makes an outbound HTTP request using params. |
| `llm` | Stub, not registered (TASK-06) | Calls OpenAI Chat Completions API with a prompt template. |
