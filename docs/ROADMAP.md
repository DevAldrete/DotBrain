# DotBrain — Development Roadmap

## Current State

The backend MVP is complete. The project can:

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

- Show any of this in a UI (frontend tasks TASK-07, TASK-08 are pending)

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
| [TASK-07](tasks/TASK-07-frontend-api-client.md) | Add typed API client (`web/src/lib/api.ts`) | Pending |
| [TASK-08](tasks/TASK-08-frontend-dashboard.md) | Build workflow dashboard pages in SvelteKit | Pending |

---

## What Is Explicitly Out of Scope (For Now)

These are valid future directions but are not part of the current plan:

- **Redis / external queue**: The goroutine approach is sufficient while the system runs as a single process. Redis becomes relevant when horizontal scaling or cross-restart durability is needed.
- **Visual drag-and-drop builder**: A proper node graph editor (e.g., using Svelte Flow) is a significant standalone effort. The JSON editor in the create workflow form is sufficient for now.
- **Real-time run streaming (SSE/WebSocket)**: Polling the run status endpoint is acceptable for the MVP dashboard.
- **Anthropic support**: The `LLMNode` architecture (once TASK-06 is done) will make adding Anthropic straightforward.
- **Database migration framework**: The `schema.sql` approach is workable while the schema is still changing. Consider `goose` or `golang-migrate` once the schema stabilizes.
- **Branching / DAG workflows**: The engine currently runs nodes in a linear sequence. Conditional branching and parallel execution are future engine improvements.
- **Authentication / authorization**: No auth exists. Add JWT or session-based auth before any public deployment.

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
```

The critical path is: **TASK-04 → TASK-01 → TASK-02 → TASK-03 → TASK-07 → TASK-08**, with TASK-05 and TASK-06 insertable after TASK-01.
