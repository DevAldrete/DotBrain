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

## 🎯 Micro-Goals & Current Tasks

Based on the core features outlined in the project README, here are structured goals. **Every goal here must start with a test.**

### 1. The "Safe Object" Validator Node (Pending)
*README Reference*: "Type-Safe Validation: Integrated 'Safe Object' nodes for schema validation and data sanitization."
- **Task**: Implement a new `SafeObjectNode` that implements `core.NodeExecutor`.
- **Requirements**:
  - Takes an `input` map and a `schema` definition (e.g., required keys, specific types).
  - Returns validated data if it matches.
  - Returns an error if validation fails.
- **TDD Start**: Write `TestSafeObjectNode_Execute_MissingField` first.

### 2. HTTP Workflow Trigger Endpoint (Pending)
- **Task**: Add a generic webhook trigger endpoint (`POST /api/v1/workflows/:id/trigger`) to `internal/api/router.go`.
- **TDD Start**: Write a test in `internal/api/router_test.go` asserting a 404 for an unknown workflow ID, and 200/202 for a successfully queued workflow execution.

### 3. Agentic AI Node (OpenAI Integration Stub)
*README Reference*: "Current: Unified interface for external LLM APIs (OpenAI, Anthropic, etc.)."
- **Task**: Create an `LLMNode` stub in `internal/core/llm_node.go`.
- **Requirements**:
  - Accepts a `prompt` string in the input map.
  - Returns a mock/stub response until the real API is integrated.
- **TDD Start**: Write `TestLLMNode_Execute_MissingPrompt` to verify validation logic.

### 4. Basic DAG / Workflow Orchestrator
*README Reference*: "Distributed workflow engine built in Go"
- **Task**: Create an `Engine` struct that strings multiple `NodeExecutor` instances together.
- **Requirements**:
  - Can register a slice of Nodes.
  - Executes them in sequence, passing output of Node A as input to Node B.
- **TDD Start**: Write `TestEngine_SequentialExecution` that passes output from an `EchoNode` to a `MathNode`.

---

## Testing Examples & Baselines Established

We have established baseline tests for:
- `internal/core/node.go` (Echo, Fail, Math nodes)
- `internal/api/router.go` (Health, Readiness, Ping)

Always run `go test ./... -v` before committing any code to ensure pristine build states.
