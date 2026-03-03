# DotBrain — Development Roadmap

## Current State (as of Phase 4)

The backend MVP is complete and the frontend dashboard exists. The project can:

- Create and store workflow definitions (POST /api/v1/workflows)
- List and retrieve workflow definitions (GET)
- Trigger a workflow run, which executes nodes in-process via a goroutine (POST /api/v1/workflows/:id/trigger)
- Return a `run_id` to the caller (HTTP 202 Accepted)
- Pass `NodeConfig.Params` to node instances at runtime (param injection)
- Record per-node execution detail (`node_executions` rows written via `DBNodeHook`)
- Correctly model run lifecycle: `pending → running → completed/failed`
- Expose run status and node details via 3 API endpoints
- Execute real outbound HTTP requests via `HttpNode`
- Execute real OpenAI Chat Completions API calls via `LLMNode`
- Mark runs as `completed` or `failed` in the database with accurate `started_at` and `completed_at` timestamps

The project **cannot** yet:

- Execute workflows as a DAG (branching, parallel nodes, conditional edges)
- Retry failed nodes automatically
- Survive a server restart without leaving runs stuck as `running`
- Cancel an in-progress run
- Authenticate any API caller
- Schedule workflows via cron
- Stream run progress in real time (UI requires a manual refresh)
- Use LLM providers other than OpenAI

---

## Phase 1 — Make Execution Actually Work

These tasks focus on the execution engine and the data it produces. They must be done before the API surface or frontend can be meaningful, because they produce the data those depend on.

| Task | File | Status |
|------|------|--------|
| [TASK-01](tasks/TASK-01-param-injection.md) | Fix param injection into node factories | **Done** |
| [TASK-02](tasks/TASK-02-node-executions-audit-trail.md) | Write `node_executions` rows during execution | **Done** |
| [TASK-04](tasks/TASK-04-fix-run-lifecycle.md) | Fix `workflow_run` status lifecycle (`pending → running`) | **Done** |

**Order dependency:** TASK-01 must be completed before TASK-05 and TASK-06, since parameterized nodes (HTTP, LLM) require param injection to work. TASK-02 requires TASK-04 to be done first so the `run_id` and `started_at` are set correctly.

---

## Phase 2 — Expand the API Surface

These tasks expose run data through HTTP endpoints. They depend on Phase 1 because without `node_executions` rows or correct run status, the endpoints return empty or misleading data.

| Task | Description | Status |
|------|-------------|--------|
| [TASK-03](tasks/TASK-03-run-queries-and-api.md) | Add missing SQL queries + new run/node API endpoints | **Done** |

---

## Phase 3 — New Node Types

With params working (TASK-01) and the API surface expanded (TASK-03), new node types can be added and immediately exposed in the dashboard.

| Task | Description | Status |
|------|-------------|--------|
| [TASK-05](tasks/TASK-05-http-node.md) | Implement `HttpNode` — outbound HTTP requests | **Done** |
| [TASK-06](tasks/TASK-06-llm-node-openai.md) | Implement `LLMNode` with real OpenAI API | **Done** |

---

## Phase 4 — Frontend Dashboard

The dashboard consumes the API surface from Phase 2. The new node types from Phase 3 make the dashboard more interesting to use, but are not strictly required.

| Task | Description | Status |
|------|-------------|--------|
| [TASK-07](tasks/TASK-07-frontend-api-client.md) | Add typed API client (`web/src/lib/api.ts`) | **Done** |
| [TASK-08](tasks/TASK-08-frontend-dashboard.md) | Build workflow dashboard pages in SvelteKit | **Done** |

---

## Phase 5 — Engine Evolution

These tasks harden and extend the core execution engine. TASK-09 (DAG) and TASK-11 (crash recovery) are independent and can be done in parallel. TASK-10 depends on TASK-09 because the retry policy is added to `NodeConfig`, which is restructured in that task.

| Task | Description | Priority | Status |
|------|-------------|----------|--------|
| [TASK-09](tasks/TASK-09-dag-edges.md) | DAG edges and branching engine | Critical | Pending |
| [TASK-10](tasks/TASK-10-retry-backoff.md) | Retry policy with exponential backoff | Critical | Pending |
| [TASK-11](tasks/TASK-11-crash-recovery.md) | Crash recovery for stale running runs | Critical | Pending |

**Order dependency:** TASK-10 requires TASK-09. TASK-11 is independent.

---

## Phase 6 — API Completeness

These tasks fill gaps in the HTTP API. Both are self-contained and can be done in any order.

| Task | Description | Priority | Status |
|------|-------------|----------|--------|
| [TASK-12](tasks/TASK-12-workflow-crud.md) | Workflow update (`PUT`) and delete (`DELETE`) endpoints | High | Pending |
| [TASK-13](tasks/TASK-13-run-cancellation.md) | Run cancellation (`POST /runs/:id/cancel`) | High | Pending |

---

## Phase 7 — Triggers

Scheduled execution via cron expressions. Depends on crash recovery (TASK-11) because the same startup repair logic applies to scheduled runs that were interrupted.

| Task | Description | Priority | Status |
|------|-------------|----------|--------|
| [TASK-14](tasks/TASK-14-cron-triggers.md) | Cron / scheduled triggers | High | Pending |

**Order dependency:** TASK-11 should be completed first.

---

## Phase 8 — Security

Authentication middleware. Self-contained and can be done at any point, but must be done before any network-accessible deployment.

| Task | Description | Priority | Status |
|------|-------------|----------|--------|
| [TASK-15](tasks/TASK-15-auth.md) | API key authentication middleware | High | Pending |

---

## Phase 9 — Node Library

Extends the LLM node to support multiple providers and removes the security problem of storing API keys in the workflow definition JSON.

| Task | Description | Priority | Status |
|------|-------------|----------|--------|
| [TASK-16](tasks/TASK-16-multi-provider-llm.md) | Multi-provider LLM node (OpenAI, Anthropic, Ollama) | Medium | Pending |

---

## Phase 10 — UX

Real-time run progress via Server-Sent Events. Additive — the polling UI continues to work while this is implemented.

| Task | Description | Priority | Status |
|------|-------------|----------|--------|
| [TASK-17](tasks/TASK-17-sse-streaming.md) | Real-time run streaming via SSE | Medium | Pending |

---

## What Is Explicitly Out of Scope (For Now)

These are valid future directions but are not part of the current plan:

- **Redis / external queue**: The goroutine approach is sufficient while the system runs as a single process. Redis becomes relevant when horizontal scaling or cross-restart durability is needed.
- **Visual drag-and-drop builder**: A proper node graph editor (e.g., using Svelte Flow) is a significant standalone effort. The JSON editor in the create workflow form is sufficient for now.
- **Database migration framework**: The `schema.sql` approach is workable while the schema is still changing. Consider `goose` or `golang-migrate` once the schema stabilizes.
- **JWT / session auth**: TASK-15 uses a simpler API-key approach. Full JWT auth is a larger effort and not required for the intended use case.

---

## Suggested Task Execution Order

```
TASK-01  (param injection)
   ├──> TASK-02  (node execution audit trail)
   │       └──> TASK-04 must be done first (run lifecycle fix)
   ├──> TASK-05  (HttpNode)
   └──> TASK-06  (LLMNode + OpenAI)

TASK-03  (run API endpoints) — can start in parallel with TASK-01
   └──> TASK-07  (frontend API client)
           └──> TASK-08  (frontend dashboard)

--- Phase 5+ ---

TASK-09  (DAG edges)          TASK-11  (crash recovery)
   └──> TASK-10  (retry)             └──> TASK-14  (cron triggers)

TASK-12  (workflow CRUD)      — independent, any time
TASK-13  (run cancellation)   — independent, any time
TASK-15  (auth)               — independent, any time
TASK-16  (multi-provider LLM) — independent, any time
TASK-17  (SSE streaming)      — independent, any time
```

The critical path for Phase 5 is: **TASK-09 → TASK-10**, with **TASK-11 → TASK-14** as a parallel track.
