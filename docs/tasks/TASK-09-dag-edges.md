# TASK-09 — DAG Edges and Branching Engine

**Phase:** 5 — Engine Evolution  
**Priority:** Critical  
**Depends on:** nothing (but most future tasks depend on this)  
**Files affected:** `internal/core/workflow.go`, `internal/core/engine.go`, `internal/core/engine_test.go`, `schema.sql` (definition column format change), `web/src/lib/types.ts`

---

## Problem

`WorkflowDefinition` is a flat `[]NodeConfig` with no `edges` field. The engine iterates the slice in order. This means:

- Every workflow is a strictly linear chain: A → B → C.
- There is no conditional routing ("if X then go to Y else go to Z").
- There is no fan-out ("run A and B in parallel after C").
- There is no fan-in ("wait for both A and B before running D").

This is the largest architectural gap. Without a real DAG, DotBrain cannot express the workflows that make n8n and Temporal valuable.

---

## Goal

Replace the linear `[]NodeConfig` execution model with a proper DAG executor. A workflow definition gains an `edges` field that describes connections between nodes. The engine resolves execution order via topological sort, runs independent nodes in parallel, and supports conditional routing via edge conditions.

---

## New Workflow Definition Format

```go
// internal/core/workflow.go

type WorkflowDefinition struct {
    Nodes []NodeConfig `json:"nodes"`
    Edges []EdgeConfig `json:"edges"`
}

type EdgeConfig struct {
    From      string `json:"from"`       // source node ID
    To        string `json:"to"`         // target node ID
    Condition string `json:"condition,omitempty"` // optional: "success" | "failure" | "" (always)
}
```

**Example — conditional routing:**

```json
{
  "nodes": [
    { "id": "fetch",        "type": "http",   "params": { "url": "https://api.example.com/data" } },
    { "id": "on-success",   "type": "llm",    "params": { "prompt": "Summarize: {{input.body}}" } },
    { "id": "on-failure",   "type": "http",   "params": { "url": "https://alerts.example.com/notify", "method": "POST" } }
  ],
  "edges": [
    { "from": "fetch", "to": "on-success", "condition": "success" },
    { "from": "fetch", "to": "on-failure", "condition": "failure" }
  ]
}
```

**Example — parallel fan-out:**

```json
{
  "nodes": [
    { "id": "start",    "type": "echo" },
    { "id": "branch-a", "type": "http", "params": { "url": "https://api-a.example.com" } },
    { "id": "branch-b", "type": "http", "params": { "url": "https://api-b.example.com" } },
    { "id": "merge",    "type": "echo" }
  ],
  "edges": [
    { "from": "start",    "to": "branch-a" },
    { "from": "start",    "to": "branch-b" },
    { "from": "branch-a", "to": "merge" },
    { "from": "branch-b", "to": "merge" }
  ]
}
```

---

## Engine Changes

### Phase 1 — Topological execution (no parallelism yet)

Replace the `[]registeredNode` slice with a DAG structure. Use Kahn's algorithm (BFS-based topological sort) for execution order.

```go
type DAGNode struct {
    id       string
    executor NodeExecutor
    deps     []string  // IDs of nodes that must complete before this one runs
    edges    []EdgeConfig
}

type Engine struct {
    nodes map[string]*DAGNode
    Hook  NodeLifecycleHook
}
```

Execution flow:
1. Build in-degree map from `edges`.
2. Start with all zero-in-degree nodes (entry points).
3. Execute each node, collect output.
4. For each outgoing edge from a completed node, check condition (`success`/`failure`/always).
5. Decrement in-degree of the target; when it reaches zero, enqueue it.
6. Repeat until the queue is empty.

**Data passing in a DAG:** Unlike the linear model where node N gets node N-1's output, a DAG node may have multiple incoming edges. The input to a node is a merged map of all its predecessors' outputs (later outputs overwrite earlier ones for conflicting keys).

### Phase 2 — Parallel execution

Nodes whose in-degree reaches zero simultaneously can run in parallel. Use a worker pool or `sync.WaitGroup` with goroutines.

```go
// Pseudo-code for parallel execution
ready := initialReadyNodes()
var wg sync.WaitGroup
for len(ready) > 0 {
    batch := ready
    ready = nil
    resultsCh := make(chan nodeResult, len(batch))
    for _, node := range batch {
        wg.Add(1)
        go func(n *DAGNode) {
            defer wg.Done()
            out, err := n.executor.Execute(ctx, mergedInput(n))
            resultsCh <- nodeResult{id: n.id, output: out, err: err}
        }(node)
    }
    wg.Wait()
    close(resultsCh)
    for result := range resultsCh {
        // process result, resolve next ready nodes
    }
}
```

---

## Backward Compatibility

Workflows with no `edges` field fall back to linear execution (edges are inferred from the node order). This keeps all existing workflow definitions working without modification.

```go
func (def *WorkflowDefinition) inferEdges() []EdgeConfig {
    if len(def.Edges) > 0 {
        return def.Edges
    }
    // Generate linear edges: nodes[0]→nodes[1]→...→nodes[N]
    edges := make([]EdgeConfig, 0, len(def.Nodes)-1)
    for i := 1; i < len(def.Nodes); i++ {
        edges = append(edges, EdgeConfig{
            From: def.Nodes[i-1].ID,
            To:   def.Nodes[i].ID,
        })
    }
    return edges
}
```

---

## Acceptance Criteria

- [ ] `WorkflowDefinition` has an `edges []EdgeConfig` field
- [ ] `EdgeConfig` has `from`, `to`, and optional `condition` fields
- [ ] Engine executes a two-node workflow with an explicit edge correctly
- [ ] Engine executes a three-node linear workflow with edges `A→B→C`
- [ ] Engine executes a fan-out: one node with two outgoing unconditional edges, both target nodes run
- [ ] Engine executes conditional routing: `condition: "success"` edge is followed on success; `condition: "failure"` edge is followed on failure; opposing branch does not run
- [ ] Cyclic graphs are detected at `LoadFromDefinition` time and return an error
- [ ] Workflows with no `edges` field execute in the same order as before (backward compat)
- [ ] `node_executions` rows are written correctly for all executed nodes
- [ ] `go test ./internal/core/...` passes

---

## TDD Approach

### Red

```go
// TestEngine_DAG_LinearEdges — explicit edges produce same result as no-edges
func TestEngine_DAG_LinearEdges(t *testing.T) { ... }

// TestEngine_DAG_FanOut — one source, two targets both execute
func TestEngine_DAG_FanOut(t *testing.T) { ... }

// TestEngine_DAG_ConditionalSuccess — success edge followed, failure edge skipped
func TestEngine_DAG_ConditionalSuccess(t *testing.T) { ... }

// TestEngine_DAG_ConditionalFailure — failure edge followed after node error
func TestEngine_DAG_ConditionalFailure(t *testing.T) { ... }

// TestEngine_DAG_CycleDetection — cyclic definition returns error at load time
func TestEngine_DAG_CycleDetection(t *testing.T) { ... }

// TestEngine_DAG_BackwardCompat — no edges field → linear execution unchanged
func TestEngine_DAG_BackwardCompat(t *testing.T) { ... }
```

---

## Definition of Done

- All acceptance criteria checked
- `go test ./...` passes with no regressions
- `web/src/lib/types.ts` updated: `WorkflowDefinition` has `edges?: EdgeConfig[]`
- `docs/core/workflow-definition.md` updated with the new format and examples
