# Engine

**Source:** `internal/core/engine.go`

The `Engine` is the central executor. It holds an ordered list of nodes and runs them sequentially, piping each node's output as the next node's input.

---

## Core Types

### `NodeExecutor` interface

```go
// internal/core/node.go
type NodeExecutor interface {
    Execute(ctx context.Context, input map[string]any) (map[string]any, error)
}
```

Every node — built-in or future — must implement this single method. The contract is:

- **Input:** a `map[string]any` (JSON object). For the first node this is the trigger payload; for all subsequent nodes it is the previous node's output.
- **Output:** a new `map[string]any` that becomes the next node's input.
- **Error:** any non-nil error aborts the entire run immediately.

### `Engine` struct

```go
type Engine struct {
    nodes []registeredNode  // ordered execution list
    Hook  NodeLifecycleHook // optional; called on start/complete/fail of each node
}
```

`Engine` is not concurrent-safe; it is constructed fresh for each workflow run inside the trigger goroutine.

---

## Node Registry

The registry maps the string `type` field from a `NodeConfig` to a factory function that instantiates the corresponding `NodeExecutor`:

```go
// internal/core/engine.go
var nodeRegistry = map[string]func(map[string]any) NodeExecutor{
    "echo":        func(p map[string]any) NodeExecutor { return EchoNode{} },
    "fail":        func(p map[string]any) NodeExecutor { return FailNode{} },
    "math":        func(p map[string]any) NodeExecutor { ... return MathNode{...} },
    "llm":         func(p map[string]any) NodeExecutor { return NewLLMNode(p) },
    "safe_object": func(p map[string]any) NodeExecutor { return SafeObjectNode{...} },
    "http":        func(p map[string]any) NodeExecutor { return NewHttpNode(p) },
}
```

The factory receives the node's `params` map (from the workflow definition JSON) and is responsible for reading all configuration out of it. **Params are bound at load time**, not at execution time — changing a param after `LoadFromDefinition` has no effect.

> **Note:** `safe_object` is registered in the registry but its type is absent from the frontend `NODE_TYPES` list in `web/src/lib/nodes.ts`, so it cannot be selected in the UI builder. It is fully functional via the API.

---

## Loading a Workflow

`LoadFromDefinition` converts a `*WorkflowDefinition` into an ordered `[]registeredNode`:

```go
func (e *Engine) LoadFromDefinition(def *WorkflowDefinition) error {
    for _, config := range def.Nodes {
        factory, exists := nodeRegistry[config.Type]
        if !exists {
            return fmt.Errorf("unknown node type: %s", config.Type)
        }
        params := config.Params
        if params == nil {
            params = map[string]any{}
        }
        e.RegisterWithID(config.ID, factory(params))
    }
    return nil
}
```

If any node type is unrecognized, the entire load fails and the run is immediately marked `failed` before any node executes.

---

## Execution Loop

```go
func (e *Engine) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
    currentData := input

    for _, nodeInfo := range e.nodes {
        // 1. Notify hook (writes node_execution row with status="running")
        if e.Hook != nil {
            e.Hook.OnNodeStart(ctx, nodeInfo.id, currentData)
        }

        // 2. Execute
        output, err := nodeInfo.executor.Execute(ctx, currentData)

        // 3a. On failure: notify hook, abort
        if err != nil {
            if e.Hook != nil {
                e.Hook.OnNodeFail(ctx, nodeInfo.id, err)
            }
            return nil, fmt.Errorf("node execution failed: %w", err)
        }

        // 3b. On success: notify hook, advance data
        if e.Hook != nil {
            e.Hook.OnNodeComplete(ctx, nodeInfo.id, output)
        }
        currentData = output  // output becomes the next node's input
    }

    return currentData, nil  // last node's output = workflow run output_data
}
```

Key behaviors:

- **Sequential only.** Nodes run one at a time, in definition order. There is no parallelism.
- **Data replacement.** The output of node N *replaces* the current data entirely — it does not merge with it. Node N+1 sees only what node N returned.
- **Fail-fast.** The first node error stops the entire run. Later nodes do not execute.
- **Context propagation.** The `context.Context` passed to `Execute` flows through to each `NodeExecutor.Execute`. Cancellations and deadlines will interrupt node I/O operations that respect context (HTTP requests, etc.).

---

## Lifecycle Hook

```go
type NodeLifecycleHook interface {
    OnNodeStart(ctx context.Context, nodeID string, input map[string]any)
    OnNodeComplete(ctx context.Context, nodeID string, output map[string]any)
    OnNodeFail(ctx context.Context, nodeID string, err error)
}
```

The hook is the bridge between the in-memory engine and the database. The production implementation is `DBNodeHook` (`internal/api/hook.go`), which writes and updates `node_executions` rows.

The hook is optional: if `Engine.Hook` is `nil`, the engine runs without any DB side effects (used in unit tests).

### `DBNodeHook` behavior

| Hook call | Database operation |
|---|---|
| `OnNodeStart` | `INSERT INTO node_executions (status='running', input_data=...)` |
| `OnNodeComplete` | `UPDATE node_executions SET status='completed', output_data=..., started_at=..., completed_at=...` |
| `OnNodeFail` | `UPDATE node_executions SET status='failed', error=..., started_at=..., completed_at=...` |

The hook stores a `map[nodeID]executionUUID` so it can correlate start and complete/fail calls without a DB round-trip.

---

## How the Engine is Wired at Runtime

The trigger handler in `internal/api/router.go` constructs and runs the engine inside a goroutine:

```go
go func(runID pgtype.UUID, w db.Workflow, initialData map[string]any) {
    ctx := context.Background()

    // 1. Transition DB run to "running"
    a.transitionToRunning(ctx, runID)

    // 2. Build engine with DB hook
    engine := core.NewEngine()
    engine.Hook = NewDBNodeHook(a.queries, runID)

    // 3. Load workflow definition
    def, err := core.ParseDefinition(w.Definition)
    if err != nil { /* mark failed */ return }

    if err := engine.LoadFromDefinition(def); err != nil { /* mark failed */ return }

    // 4. Execute
    output, err := engine.Execute(ctx, initialData)
    if err != nil { /* mark failed */ return }

    // 5. Mark completed
    a.updateRunStatus(ctx, runID, "completed", output, "")
}(pgRunID, workflow, payload)
```

The trigger endpoint returns HTTP 202 immediately after spawning the goroutine; the caller uses the returned `run_id` to poll for status.

---

## Adding a New Node Type

1. Implement `NodeExecutor` in a new file under `internal/core/`.
2. Register the type in `nodeRegistry` in `engine.go`.
3. Add its metadata to `NODE_TYPES` in `web/src/lib/nodes.ts` so it appears in the UI builder.
4. Add its `NodeType` literal to the `NodeType` union in `web/src/lib/types.ts`.
