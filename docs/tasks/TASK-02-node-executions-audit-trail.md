# TASK-02 — Write `node_executions` Audit Trail During Execution

**Phase:** 1 — Make Execution Actually Work  
**Priority:** High  
**Depends on:** TASK-01 (param injection), TASK-04 (run lifecycle fix)  
**Files affected:** `internal/core/engine.go`, `internal/api/router.go`

---

## Problem

The `node_executions` table exists in the schema and the generated sqlc queries exist (`CreateNodeExecution`, `UpdateNodeExecutionStatus`), but they are never called. Every workflow run executes its nodes without leaving any per-step record.

This means:
- There is no way to see which step of a workflow failed and why
- The dashboard (TASK-08) cannot show node-level detail
- Debugging a failed workflow requires reading logs, not querying the DB

The `Engine.Execute` method has no access to the database, which is by design (the engine is a pure orchestration layer). The bridge between the engine and the DB must be solved architecturally.

---

## Goal

Every node execution produces a `node_executions` row with accurate `status`, `input_data`, `output_data`, `error`, `started_at`, and `completed_at` fields.

---

## Design Decision: How to Give the Engine DB Access

Two options:

**Option A — Callbacks on Engine**

The engine accepts an optional `NodeLifecycleHook` interface. The API layer wires up a DB-backed implementation. The engine remains testable without a DB.

```go
type NodeLifecycleHook interface {
    OnNodeStart(ctx context.Context, nodeID string, input map[string]any)
    OnNodeComplete(ctx context.Context, nodeID string, output map[string]any)
    OnNodeFail(ctx context.Context, nodeID string, err error)
}
```

**Option B — Pass runID to Execute**

The engine's `Execute` method accepts a `runID` and a `Queries` pointer and writes records directly.

Option A is preferred because it keeps the engine decoupled from the database and makes unit testing straightforward.

---

## Acceptance Criteria

- [ ] `Engine` has a `Hook NodeLifecycleHook` field (can be nil — no-op if not set)
- [ ] Before each node's `Execute`, `Hook.OnNodeStart` is called with the node ID and input
- [ ] After each node's `Execute` succeeds, `Hook.OnNodeComplete` is called with the output
- [ ] After each node's `Execute` fails, `Hook.OnNodeFail` is called with the error
- [ ] The API layer (`workflowTriggerHandler`) creates a `DBNodeHook` that implements `NodeLifecycleHook` using `*db.Queries` and the `runID`
- [ ] Each hook call writes/updates a `node_executions` row correctly
- [ ] When `Hook` is nil, the engine runs normally without panicking
- [ ] All existing tests pass (they don't set a hook)
- [ ] New tests cover hook invocation order and DB writes

---

## TDD Approach

### Red — write failing tests first

**File:** `internal/core/engine_test.go`

```go
// TestEngine_Execute_CallsHookForEachNode verifies that the lifecycle hook
// is called once per node in the correct order.
func TestEngine_Execute_CallsHookForEachNode(t *testing.T) {
    hook := &recordingHook{}
    engine := NewEngine()
    engine.Hook = hook
    engine.Register(EchoNode{})
    engine.Register(EchoNode{})

    input := map[string]any{"key": "value"}
    _, err := engine.Execute(context.Background(), input)
    if err != nil {
        t.Fatal(err)
    }

    if len(hook.starts) != 2 {
        t.Errorf("expected 2 OnNodeStart calls, got %d", len(hook.starts))
    }
    if len(hook.completes) != 2 {
        t.Errorf("expected 2 OnNodeComplete calls, got %d", len(hook.completes))
    }
}

// TestEngine_Execute_CallsOnNodeFail verifies that OnNodeFail is called
// when a node returns an error.
func TestEngine_Execute_CallsOnNodeFail(t *testing.T) {
    hook := &recordingHook{}
    engine := NewEngine()
    engine.Hook = hook
    engine.Register(FailNode{})

    _, err := engine.Execute(context.Background(), map[string]any{})
    if err == nil {
        t.Fatal("expected error from FailNode")
    }
    if len(hook.failures) != 1 {
        t.Errorf("expected 1 OnNodeFail call, got %d", len(hook.failures))
    }
}

// TestEngine_Execute_NilHookDoesNotPanic verifies that an engine with
// no hook set runs normally.
func TestEngine_Execute_NilHookDoesNotPanic(t *testing.T) {
    engine := NewEngine()
    engine.Register(EchoNode{})
    _, err := engine.Execute(context.Background(), map[string]any{"x": 1})
    if err != nil {
        t.Fatal(err)
    }
}
```

### Green — minimal implementation

1. Define the interface and add the field to `Engine`:
   ```go
   type NodeLifecycleHook interface {
       OnNodeStart(ctx context.Context, nodeID string, input map[string]any)
       OnNodeComplete(ctx context.Context, nodeID string, output map[string]any)
       OnNodeFail(ctx context.Context, nodeID string, err error)
   }

   type Engine struct {
       nodes []NodeExecutor
       Hook  NodeLifecycleHook
   }
   ```

2. Update `Execute` to call the hook:
   ```go
   for _, node := range e.nodes {
       if e.Hook != nil {
           e.Hook.OnNodeStart(ctx, node.ID(), currentData)
       }
       output, err := node.Execute(ctx, currentData)
       if err != nil {
           if e.Hook != nil {
               e.Hook.OnNodeFail(ctx, node.ID(), err)
           }
           return nil, fmt.Errorf("node execution failed: %w", err)
       }
       if e.Hook != nil {
           e.Hook.OnNodeComplete(ctx, node.ID(), output)
       }
       currentData = output
   }
   ```

   > Note: This requires nodes to expose their ID. The engine needs to store `(id, executor)` pairs rather than bare `NodeExecutor` slices. This is the "refactor" noted in TASK-01.

3. Implement `DBNodeHook` in `internal/api/`:
   ```go
   type DBNodeHook struct {
       queries *db.Queries
       runID   pgtype.UUID
   }
   ```

### Refactor

- Consider whether `node.ID()` belongs on the `NodeExecutor` interface or whether the engine tracks IDs separately in a `[]struct{ id string; exec NodeExecutor }` slice. The latter keeps the interface minimal.

---

## Definition of Done

- `go test ./...` passes
- Triggering a workflow and letting it complete produces `node_executions` rows in the DB
- Each row has accurate `status`, `input_data`, `output_data`, `started_at`, and `completed_at`
- A failed node produces a row with `status = "failed"` and a non-empty `error` field
