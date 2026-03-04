# Gaps — What DotBrain Lacks

This document is an honest accounting of what the system cannot do yet, grouped by severity. Each gap links to a task with a concrete implementation plan.

---

## Severity Guide

| Level | Meaning |
|---|---|
| **Critical** | Prevents the system from being usable as a workflow engine in any real scenario |
| **High** | Core feature expected from any n8n/Temporal-like system; absence limits adoption significantly |
| **Medium** | Quality-of-life or operational necessity; required before any public deployment |
| **Low** | Valuable improvements; safe to defer |

---

## Critical Gaps

### 1. No DAG — only linear pipelines

**Task:** [TASK-09](../tasks/TASK-09-dag-edges.md)

`WorkflowDefinition` is a flat `[]NodeConfig`. There are no edges. Every workflow is a strict linear chain: A → B → C.

Real-world automation almost always requires:
- **Conditional routing** — "if status_code == 200, go to node X; else go to node Y"
- **Fan-out** — run two HTTP calls in parallel
- **Fan-in** — wait for multiple branches before continuing
- **Loops** — retry or iterate over a list

Without edges, DotBrain can only express the simplest pipelines. This is the single largest architectural gap.

**Proposed solution:** Add an `edges` field to `WorkflowDefinition`. Change the engine from a `[]registeredNode` slice to a DAG executor that resolves execution order via topological sort and supports parallel branch execution with goroutines + `sync.WaitGroup`.

---

### 2. No retry or backoff

**Task:** [TASK-10](../tasks/TASK-10-retry-backoff.md)

If a node fails (network timeout, API rate limit, transient error), the entire run immediately fails. There is no retry logic anywhere.

The `retrying` status already exists in `node_executions.status` and the DB schema, but nothing ever sets it.

Temporal's core value proposition is durable, retryable execution. Without retries, DotBrain cannot reliably run workflows that touch external APIs.

**Proposed solution:** Add `retry_policy` to `NodeConfig` (max attempts, backoff strategy). The engine loop checks the policy on error and re-executes the node with exponential backoff before marking it failed.

---

### 3. Stale `running` runs after a crash

**Task:** [TASK-11](../tasks/TASK-11-crash-recovery.md)

Workflow executions run in goroutines with no recovery mechanism. If the server process crashes or is restarted (e.g., a deployment), any in-flight runs remain in `status = 'running'` forever. There is no watchdog, no timeout, and no recovery path.

This is a correctness issue: the run history becomes permanently misleading, and callers polling for completion will wait indefinitely.

**Proposed solution:** On startup, scan `workflow_runs` for rows stuck in `running` and mark them `failed` with an error message indicating a server restart. Add a periodic watchdog that does the same for runs exceeding a configurable max duration.

---

## High Gaps

### 4. No workflow update or delete

**Task:** [TASK-12](../tasks/TASK-12-workflow-crud.md)

The API only has `POST /workflows` and `GET /workflows/:id`. There is no way to edit a workflow definition or delete a workflow through the API. Any change requires direct database manipulation.

**Proposed solution:** `PUT /api/v1/workflows/:id` to replace the definition, and `DELETE /api/v1/workflows/:id` which cascade-deletes runs and node executions.

---

### 5. No run cancellation

**Task:** [TASK-13](../tasks/TASK-13-run-cancellation.md)

Once triggered, a run cannot be stopped. The `cancelled` status exists in the DB schema but is never set. Long-running nodes (slow HTTP calls, expensive LLM requests) cannot be interrupted.

**Proposed solution:** Store a `context.CancelFunc` per active run in a protected map. `POST /api/v1/runs/:id/cancel` calls the cancel function, which propagates through the `context.Context` to all node I/O operations. The engine catches the cancellation and marks the run `cancelled`.

---

### 6. No authentication

**Task:** [TASK-15](../tasks/TASK-15-auth.md)

Every endpoint is publicly accessible with no authentication or authorization. Anyone who can reach the server can create workflows, trigger runs, and read all execution data including LLM prompts and API responses stored in `node_executions.input_data`.

**Proposed solution:** API key authentication via a static `Authorization: Bearer <key>` header, validated by a Gin middleware. The key is configured via environment variable. This is the minimum viable gate before any network-accessible deployment.

---

### 7. No scheduling or cron triggers

**Task:** [TASK-14](../tasks/TASK-14-cron-triggers.md)

Workflows can only be triggered manually via `POST /workflows/:id/trigger`. There is no way to run a workflow on a schedule (every hour, every day at 9am, etc.) — which is one of n8n's most used features.

**Proposed solution:** Add a `schedules` table with cron expressions. A background goroutine (using a cron library like `robfig/cron`) fires triggers at the scheduled time by calling the same internal trigger logic used by the HTTP handler.

---

## Medium Gaps

### 8. No real-time run status

**Task:** [TASK-17](../tasks/TASK-17-sse-streaming.md)

The UI refreshes run status with a manual "Refresh" button. There is no push mechanism. Watching a long-running workflow requires repeated polling by the user.

**Proposed solution:** Server-Sent Events (SSE) endpoint `GET /api/v1/runs/:id/stream`. The engine's `NodeLifecycleHook` publishes events to a per-run channel; the SSE handler fans them out to connected clients. SSE is simpler than WebSockets and sufficient for one-directional status updates.

---

### 9. Multi-provider LLM

**Task:** [TASK-16](../tasks/TASK-16-multi-provider-llm.md)

`LLMNode` is hardcoded to `https://api.openai.com`. The README promises a "unified interface for OpenAI, Anthropic, etc." but only OpenAI is implemented. Anthropic, Google Gemini, and local models (Ollama) are absent.

**Proposed solution:** Introduce a `provider` param (`"openai"`, `"anthropic"`, `"ollama"`). Each provider implements a common internal `LLMProvider` interface. `LLMNode` delegates to the correct provider at execution time.

---

### 10. API pagination

No list endpoint (`GET /workflows`, `GET /workflows/:id/runs`) accepts pagination parameters. Both are hardcoded to `LIMIT 100`. This will silently truncate data once a user has more than 100 workflows or runs.

**Fix:** Accept `limit` and `offset` (or `cursor`) query parameters. Low effort; can be done in any PR touching the list handlers.

---

## Dependency Graph for New Tasks

```
TASK-09 (DAG engine)        ← foundational; most other features build on top
    └──> TASK-10 (retries)  ← retry policy needs per-node config in NodeConfig

TASK-11 (crash recovery)    ← independent; startup + watchdog only

TASK-12 (CRUD)              ← independent; pure API work
TASK-13 (cancellation)      ← independent; requires context map in router
TASK-15 (auth)              ← independent; Gin middleware only

TASK-14 (cron)              ← depends on stable trigger logic (currently fine)
TASK-17 (SSE)               ← independent; new endpoint + hook channel

TASK-16 (multi-LLM)         ← independent; isolated to llm_node.go
```

---

## What Is Explicitly Out of Scope

- **External task queue (Redis/NATS):** The goroutine-per-run model works correctly for a single-process deployment. A queue becomes necessary only when horizontal scaling is needed. Defer until after TASK-09 and TASK-11 are done and the single-node model is proven stable.
- **Visual drag-and-drop designer:** A canvas editor (Svelte Flow / xyflow) is a large standalone effort. The JSON-based workflow builder in the current UI is sufficient for developers. Defer until TASK-09 (DAG) is complete, since the designer needs edges to be meaningful.
- **Database migration framework:** `schema.sql` direct-apply is fine while the schema is still changing. Adopt `goose` or `golang-migrate` once the schema stabilizes post-TASK-09.
- **Workflow versioning:** Storing multiple versions of a definition and associating runs with a specific version. Valid future need; not urgent.
