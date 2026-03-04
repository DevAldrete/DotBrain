# Dotbrain Backend Tasks & Testing Guide

This document defines the approach for backend development in the Dotbrain project and outlines specific micro-goals and issues.

## Testing Philosophy

As stated in our core principles, we strictly adhere to **Test-Driven Development (TDD)** using the Red-Green-Refactor cycle:

> **The Iron Law:** NO PRODUCTION CODE WITHOUT A FAILING TEST FIRST

### Rules of Engagement
1. **Red**: Write a clear, descriptive failing test demonstrating the behavior (or reproducing a bug).
2. **Verify Red**: Watch it fail (e.g., `go test ./...`) for the right reason (e.g. function missing, logical error).
3. **Green**: Write the minimal code in Go to make the test pass. Do not over-engineer.
4. **Verify Green**: Ensure all tests pass.
5. **Refactor**: Clean up duplication, improve names, extract interfaces if necessary.

For our Go backend, use standard library `testing` with table-driven tests when multiple conditions need evaluation. Use `net/http/httptest` for HTTP handlers.

---

## Current Tasks

All backend tasks (TASK-01 through TASK-06) are **complete**. Frontend tasks (TASK-07, TASK-08) are pending. See `docs/ROADMAP.md` for the full plan.

| Task | File | Status |
|------|------|--------|
| TASK-01 | [docs/tasks/TASK-01-param-injection.md](tasks/TASK-01-param-injection.md) | **Done** |
| TASK-02 | [docs/tasks/TASK-02-node-executions-audit-trail.md](tasks/TASK-02-node-executions-audit-trail.md) | **Done** |
| TASK-03 | [docs/tasks/TASK-03-run-queries-and-api.md](tasks/TASK-03-run-queries-and-api.md) | **Done** |
| TASK-04 | [docs/tasks/TASK-04-fix-run-lifecycle.md](tasks/TASK-04-fix-run-lifecycle.md) | **Done** |
| TASK-05 | [docs/tasks/TASK-05-http-node.md](tasks/TASK-05-http-node.md) | **Done** |
| TASK-06 | [docs/tasks/TASK-06-llm-node-openai.md](tasks/TASK-06-llm-node-openai.md) | **Done** |
| TASK-07 | [docs/tasks/TASK-07-frontend-api-client.md](tasks/TASK-07-frontend-api-client.md) | Pending |
| TASK-08 | [docs/tasks/TASK-08-frontend-dashboard.md](tasks/TASK-08-frontend-dashboard.md) | Pending |

---

## Completed Micro-Goals (Original)

The tasks below are done and represent the baseline state of the project.

### 1. The "Safe Object" Validator Node
- `SafeObjectNode` is implemented in `internal/core/safe_object_node.go`
- Not yet registered in the engine registry (see TASK-01 for the registration pattern)

### 2. HTTP Workflow Trigger Endpoint
- `POST /api/v1/workflows/:id/trigger` is implemented in `internal/api/router.go`
- Returns 202 Accepted with a `run_id`

### 3. Agentic AI Node
- `LLMNode` is implemented in `internal/core/llm_node.go`
- Calls the OpenAI Chat Completions API using raw `net/http` (no external library)
- Registered as `"llm"` in the engine node registry

### 4. Basic DAG / Workflow Orchestrator
- `Engine` is implemented in `internal/core/engine.go`
- Sequential execution with `EchoNode`, `FailNode`, `MathNode`

---

## Testing Examples & Baselines Established

We have established baseline tests for:
- `internal/core/node.go` (Echo, Fail, Math nodes)
- `internal/core/http_node.go` (HttpNode — 9 tests using httptest.NewServer)
- `internal/core/llm_node.go` (LLMNode — 6 tests using mock OpenAI server)
- `internal/api/router.go` (Health, Readiness, Ping, Trigger, run lifecycle, 3 run/node endpoints)
- `internal/api/hook.go` (DBNodeHook lifecycle callbacks)

**33 tests total, all passing.** 2 skipped (DB-dependent legacy tests).

Always run `go test ./... -v` before committing any code to ensure pristine build states.
