# TASK-01 — Fix Param Injection into Node Factories

**Phase:** 1 — Make Execution Actually Work  
**Priority:** Critical (blocks TASK-05 and TASK-06)  
**Files affected:** `internal/core/engine.go`, `internal/core/node.go`, `internal/core/llm_node.go`, `internal/core/safe_object_node.go`

---

## Problem

`NodeConfig.Params` are currently discarded at engine load time. The node registry stores zero-argument factory functions:

```go
// internal/core/engine.go (current)
var nodeRegistry = map[string]func() NodeExecutor{
    "echo": func() NodeExecutor { return EchoNode{} },
    "math": func() NodeExecutor { return MathNode{} },
}
```

When `LoadFromDefinition` iterates over nodes, it calls `factory()` and throws away `config.Params`:

```go
// internal/core/engine.go (current)
e.Register(factory())  // config.Params is never passed
```

This means no node can be configured at definition time. An `LLMNode` with `"params": {"prompt": "Summarize: {{input.text}}"}` would silently ignore the prompt template.

---

## Goal

Every node factory receives `map[string]any` params so nodes can configure themselves at instantiation. Nodes that do not need params simply ignore the argument.

---

## Acceptance Criteria

- [x] The node registry signature changes from `map[string]func() NodeExecutor` to `map[string]func(map[string]any) NodeExecutor`
- [x] `LoadFromDefinition` passes `config.Params` (or an empty map if nil) to each factory
- [x] `EchoNode` and `FailNode` compile and pass tests with the new signature (they ignore params)
- [x] `MathNode` optionally reads `a` and `b` from params as defaults, falling back to input values
- [x] All existing tests in `internal/core/` continue to pass without modification (backwards-compatible behavior)
- [x] No production code is written without a failing test first (TDD)

> **Status: COMPLETE** — This task was already implemented before the current development session began. The registry already used `func(map[string]any) NodeExecutor` and `LoadFromDefinition` already passed `config.Params`.

---

## TDD Approach

### Red — write failing tests first

**File:** `internal/core/engine_test.go`

```go
// TestEngine_LoadFromDefinition_PassesParamsToNode verifies that params
// defined in a NodeConfig are passed to the node at instantiation time.
func TestEngine_LoadFromDefinition_PassesParamsToNode(t *testing.T) {
    // A "capture" node that records the params it received.
    // Register it in a test-local registry override.
    // Assert that after LoadFromDefinition, the node holds the correct params.
}

// TestEngine_LoadFromDefinition_NilParamsSafe verifies that a NodeConfig
// with no params field does not panic (passes an empty map instead of nil).
func TestEngine_LoadFromDefinition_NilParamsSafe(t *testing.T) {
    def := &WorkflowDefinition{
        Nodes: []NodeConfig{
            {ID: "1", Type: "echo"}, // no Params field
        },
    }
    engine := NewEngine()
    err := engine.LoadFromDefinition(def)
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
}
```

### Green — minimal implementation

1. Change the registry type:
   ```go
   var nodeRegistry = map[string]func(map[string]any) NodeExecutor{
       "echo": func(p map[string]any) NodeExecutor { return EchoNode{} },
       "fail": func(p map[string]any) NodeExecutor { return FailNode{} },
       "math": func(p map[string]any) NodeExecutor { return MathNode{} },
   }
   ```

2. Update `LoadFromDefinition`:
   ```go
   params := config.Params
   if params == nil {
       params = map[string]any{}
   }
   e.Register(factory(params))
   ```

### Refactor

- Consider whether `Register` should accept a `NodeConfig` instead of a bare `NodeExecutor`, which would allow the engine to track node IDs (needed later for TASK-02).

---

## Definition of Done

- `go test ./internal/core/...` passes with the new registry signature
- `go test ./...` (full suite) passes — no regressions in API tests
- The comment on line 34 of `engine.go` ("For a more advanced setup...") is removed and replaced with real code
