# Workflow Definition

**Source:** `internal/core/workflow.go`

A workflow definition is the static description of what a workflow does. It is stored as a JSONB document in the `workflows.definition` column and parsed into Go structs at execution time.

---

## Go Types

```go
// internal/core/workflow.go

type WorkflowDefinition struct {
    Nodes []NodeConfig `json:"nodes"`
}

type NodeConfig struct {
    ID     string         `json:"id"`
    Type   string         `json:"type"`
    Params map[string]any `json:"params,omitempty"`
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
      "params": { }
    }
  ]
}
```

### Field reference

| Field | Type | Required | Rules |
|---|---|---|---|
| `nodes` | array | yes | Ordered list of steps. Executed top-to-bottom. Must not be empty. |
| `nodes[].id` | string | yes | Must be unique within the workflow. Used as the `node_id` in `node_executions` rows. |
| `nodes[].type` | string | yes | Must match a key in the engine's node registry: `echo`, `fail`, `math`, `http`, `llm`, `safe_object`. |
| `nodes[].params` | object | no | Node-specific configuration. See [nodes.md](nodes.md) for each type's param schema. Omit or set to `{}` for nodes with no params. |

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
- **No edges yet:** the current engine only supports a linear `nodes` array. There is no `edges` field. Branching, fan-out, and conditional routing are not yet implemented.
