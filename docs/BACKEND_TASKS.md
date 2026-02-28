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

The original micro-goals below have been completed. Active development is now tracked in the task files under `docs/tasks/`. See `docs/ROADMAP.md` for the full plan and execution order.

| Task | File | Description |
|------|------|-------------|
| TASK-01 | [docs/tasks/TASK-01-param-injection.md](tasks/TASK-01-param-injection.md) | Fix param injection into node factories |
| TASK-02 | [docs/tasks/TASK-02-node-executions-audit-trail.md](tasks/TASK-02-node-executions-audit-trail.md) | Write `node_executions` rows during execution |
| TASK-03 | [docs/tasks/TASK-03-run-queries-and-api.md](tasks/TASK-03-run-queries-and-api.md) | Add missing SQL queries and run API endpoints |
| TASK-04 | [docs/tasks/TASK-04-fix-run-lifecycle.md](tasks/TASK-04-fix-run-lifecycle.md) | Fix workflow run status lifecycle |
| TASK-05 | [docs/tasks/TASK-05-http-node.md](tasks/TASK-05-http-node.md) | Implement `HttpNode` |
| TASK-06 | [docs/tasks/TASK-06-llm-node-openai.md](tasks/TASK-06-llm-node-openai.md) | Implement `LLMNode` with real OpenAI API |
| TASK-07 | [docs/tasks/TASK-07-frontend-api-client.md](tasks/TASK-07-frontend-api-client.md) | Add typed frontend API client |
| TASK-08 | [docs/tasks/TASK-08-frontend-dashboard.md](tasks/TASK-08-frontend-dashboard.md) | Build workflow dashboard in SvelteKit |

---

## Completed Micro-Goals (Original)

The tasks below are done and represent the baseline state of the project.

### 1. The "Safe Object" Validator Node
- `SafeObjectNode` is implemented in `internal/core/safe_object_node.go`
- Not yet registered in the engine registry (see TASK-01 for the registration pattern)

### 2. HTTP Workflow Trigger Endpoint
- `POST /api/v1/workflows/:id/trigger` is implemented in `internal/api/router.go`
- Returns 202 Accepted with a `run_id`

### 3. Agentic AI Node (Stub)
- `LLMNode` stub is implemented in `internal/core/llm_node.go`
- Not yet wired to OpenAI (see TASK-06)

### 4. Basic DAG / Workflow Orchestrator
- `Engine` is implemented in `internal/core/engine.go`
- Sequential execution with `EchoNode`, `FailNode`, `MathNode`

---

## Testing Examples & Baselines Established

We have established baseline tests for:
- `internal/core/node.go` (Echo, Fail, Math nodes)
- `internal/api/router.go` (Health, Readiness, Ping)

Always run `go test ./... -v` before committing any code to ensure pristine build states.
