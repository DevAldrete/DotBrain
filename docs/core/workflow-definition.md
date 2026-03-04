# Workflow Definition

**Source:** `internal/core/workflow.go`

A workflow definition is the static description of what a workflow does. It is stored as a JSONB document in the `workflows.definition` column and parsed into Go structs at execution time.

---

## Go Types

```go
// internal/core/workflow.go

type WorkflowDefinition struct {
    Nodes []NodeConfig `json:"nodes"`
    Edges []EdgeConfig `json:"edges"`
}

type EdgeConfig struct {
    From      string `json:"from"`                // source node ID
    To        string `json:"to"`                  // target node ID
    Condition string `json:"condition,omitempty"` // "success" | "failure" | "" (always)
}

type NodeConfig struct {
    ID          string         `json:"id"`
    Type        string         `json:"type"`
    Params      map[string]any `json:"params,omitempty"`
    RetryPolicy *RetryPolicy   `json:"retry_policy,omitempty"`
}

type RetryPolicy struct {
    MaxAttempts     int     `json:"max_attempts"`        // total attempts including the first; default 1 (no retry)
    InitialInterval int     `json:"initial_interval_ms"` // milliseconds; default 1000
    BackoffFactor   float64 `json:"backoff_factor"`      // multiplier per attempt; default 2.0
    MaxInterval     int     `json:"max_interval_ms"`     // cap on backoff; default 30000 (30s)
}
```

---

## JSON Format

```json
{
  "nodes": [
    {
      "id": "<unique-string>",
      "type": "<node-type>",
      "params": { },
      "retry_policy": {
        "max_attempts": 3,
        "initial_interval_ms": 1000,
        "backoff_factor": 2.0,
        "max_interval_ms": 30000
      }
    }
  ],
  "edges": [
    {
      "from": "<source-node-id>",
      "to": "<target-node-id>",
      "condition": "success"
    }
  ]
}
```

### Field reference

| Field | Type | Required | Rules |
|---|---|---|---|
| `nodes` | array | yes | List of steps. Must not be empty. |
| `nodes[].id` | string | yes | Must be unique within the workflow. Used as the `node_id` in `node_executions` rows. |
| `nodes[].type` | string | yes | Must match a key in the engine's node registry: `echo`, `fail`, `math`, `http`, `llm`, `safe_object`. |
| `nodes[].params` | object | no | Node-specific configuration. See [nodes.md](nodes.md) for each type's param schema. Omit or set to `{}` for nodes with no params. |
| `nodes[].retry_policy` | object | no | Retry configuration for this node on failure. If omitted, the node runs once (no retry). See below for fields. |
| `nodes[].retry_policy.max_attempts` | int | yes | Total attempts including the initial try. `3` means 1 initial + 2 retries. |
| `nodes[].retry_policy.initial_interval_ms` | int | yes | Backoff delay in milliseconds before the first retry. |
| `nodes[].retry_policy.backoff_factor` | float | yes | Multiplier applied per retry. `2.0` = exponential doubling. `1.0` = constant interval. |
| `nodes[].retry_policy.max_interval_ms` | int | yes | Upper bound on the backoff delay. Prevents unbounded wait times. |
| `edges` | array | no | Connections between nodes forming a DAG. If omitted, edges are inferred from node order (linear execution). |
| `edges[].from` | string | yes | Source node ID. Must match a `nodes[].id`. |
| `edges[].to` | string | yes | Target node ID. Must match a `nodes[].id`. |
| `edges[].condition` | string | no | `"success"` (follow on success), `"failure"` (follow on error), or omit/`""` for unconditional. |

### Constraints

- `nodes` must contain at least one entry (enforced by the engine at load time).
- `id` values must be unique — the `node_executions` table has a `UNIQUE(workflow_run_id, node_id)` constraint.
- Unknown `type` values cause `LoadFromDefinition` to return an error and the run to be immediately marked `failed`.
- The definition is validated structurally at parse time (`ParseDefinition`), but `type` validity and param correctness are only checked when `LoadFromDefinition` runs (at trigger time, not at workflow creation time).

---

## Examples

### Minimal: single echo node

```json
{
  "nodes": [
    { "id": "passthrough", "type": "echo" }
  ]
}
```

Trigger with any JSON payload → the same payload comes back as `output_data`.

---

### HTTP → LLM pipeline

Fetches content from an API, then summarizes it with GPT.

```json
{
  "nodes": [
    {
      "id": "fetch-content",
      "type": "http",
      "params": {
        "url": "https://api.example.com/articles/{{input.article_id}}",
        "method": "GET"
      }
    },
    {
      "id": "summarize",
      "type": "llm",
      "params": {
        "prompt": "Summarize this article in 3 bullet points:\n\n{{input.body}}",
        "model": "gpt-4o-mini",
        "max_tokens": 200,
        "api_key": "sk-..."
      }
    }
  ]
}
```

**Data flow:**

```
trigger payload: { "article_id": "42" }
    │
    ▼ fetch-content
input:  { "article_id": "42" }
output: { "status_code": 200, "body": "..article text..", "headers": {...} }
    │
    ▼ summarize
input:  { "status_code": 200, "body": "..article text..", "headers": {...} }
         ↳ {{input.body}} resolves to the article text
output: { "response": "• Point 1\n• Point 2\n• Point 3", "model": "gpt-4o-mini", ... }
    │
    ▼ workflow_run.output_data
{ "response": "• Point 1\n• Point 2\n• Point 3", "model": "gpt-4o-mini", ... }
```

---

### Validation pipeline: math → safe_object

```json
{
  "nodes": [
    {
      "id": "compute",
      "type": "math",
      "params": { "b": 100 }
    },
    {
      "id": "validate",
      "type": "safe_object",
      "params": {
        "schema": { "result": "float64" }
      }
    }
  ]
}
```

Trigger with `{"a": 42}` → `compute` returns `{"result": 142}` → `validate` checks it and passes it through.

---

## Parsing

`ParseDefinition` is called at trigger time, immediately before `LoadFromDefinition`:

```go
func ParseDefinition(data []byte) (*WorkflowDefinition, error) {
    var def WorkflowDefinition
    if err := json.Unmarshal(data, &def); err != nil {
        return nil, err
    }
    return &def, nil
}
```

The raw `[]byte` comes from the `workflows.definition` JSONB column (retrieved via sqlc). JSON unmarshal maps `params` to `map[string]any`, so all JSON numbers are `float64` in Go — node factories must type-assert accordingly.

---

## Authoring Tips

- **ID naming:** use descriptive kebab-case IDs (`fetch-user`, `summarize-content`) — they appear in the UI and in `node_executions` records, making run logs readable.
- **Param vs. input:** params are static (set at definition time); use `{{input.field}}` substitution when a value should come from the runtime data stream.
- **Edges:** use `edges` to define DAG structures. Omit `edges` for simple linear pipelines — the engine infers linear edges from node order.
- **Conditions:** use `"condition": "success"` or `"condition": "failure"` on edges for conditional routing. Omit `condition` for unconditional edges.
- **Cycles:** the engine rejects cyclic graphs at load time. All DAGs must be acyclic.

---

## DAG Examples

### Conditional routing

Route to different nodes based on success or failure of a predecessor:

```json
{
  "nodes": [
    { "id": "fetch", "type": "http", "params": { "url": "https://api.example.com/data" } },
    { "id": "on-success", "type": "llm", "params": { "prompt": "Summarize: {{input.body}}" } },
    { "id": "on-failure", "type": "http", "params": { "url": "https://alerts.example.com/notify", "method": "POST" } }
  ],
  "edges": [
    { "from": "fetch", "to": "on-success", "condition": "success" },
    { "from": "fetch", "to": "on-failure", "condition": "failure" }
  ]
}
```

**Data flow:**

```
trigger payload → fetch
                     ├─ success → on-success (receives fetch output)
                     └─ failure → on-failure (receives original input)
```

---

### Parallel fan-out

Run multiple nodes in parallel after a common predecessor:

```json
{
  "nodes": [
    { "id": "start", "type": "echo" },
    { "id": "branch-a", "type": "http", "params": { "url": "https://api-a.example.com" } },
    { "id": "branch-b", "type": "http", "params": { "url": "https://api-b.example.com" } },
    { "id": "merge", "type": "echo" }
  ],
  "edges": [
    { "from": "start", "to": "branch-a" },
    { "from": "start", "to": "branch-b" },
    { "from": "branch-a", "to": "merge" },
    { "from": "branch-b", "to": "merge" }
  ]
}
```

**Data flow:**

```
trigger payload → start
                    ├─ branch-a (parallel)
                    └─ branch-b (parallel)
                         │
                         ▼
                       merge (waits for both, receives merged outputs)
```

Nodes at the same depth with zero remaining in-degree execute concurrently.

---

### Retry with exponential backoff

Retry a flaky HTTP call up to 5 times with exponential backoff (1s, 2s, 4s, 8s) capped at 10s:

```json
{
  "nodes": [
    {
      "id": "fetch-data",
      "type": "http",
      "params": {
        "url": "https://api.example.com/unstable-endpoint",
        "method": "GET"
      },
      "retry_policy": {
        "max_attempts": 5,
        "initial_interval_ms": 1000,
        "backoff_factor": 2.0,
        "max_interval_ms": 10000
      }
    },
    {
      "id": "process",
      "type": "llm",
      "params": { "prompt": "Summarize: {{input.body}}" }
    }
  ],
  "edges": [
    { "from": "fetch-data", "to": "process" }
  ]
}
```

**Behavior:**

- Attempt 1 fails → wait 1s → attempt 2 fails → wait 2s → attempt 3 succeeds → continue to `process`
- If all 5 attempts fail, the node is marked `failed` and the `OnNodeFail` hook fires
- The `OnNodeRetry` lifecycle hook fires before each retry wait, enabling status tracking
- Context cancellation (e.g., workflow timeout) aborts the retry wait immediately
