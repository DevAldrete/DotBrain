# Nodes

**Source:** `internal/core/node.go`, `http_node.go`, `llm_node.go`, `safe_object_node.go`

All nodes implement the same interface:

```go
type NodeExecutor interface {
    Execute(ctx context.Context, input map[string]any) (map[string]any, error)
}
```

Params are injected at **load time** (when `Engine.LoadFromDefinition` is called). The `input` map at **execution time** is the previous node's output (or the trigger payload for the first node).

For nodes that support `{{input.field}}` substitution, the substitution is applied to string param values using the execution-time input — so a param like `"https://api.example.com/{{input.id}}"` resolves dynamically.

---

## Template Substitution

**Source:** `internal/core/http_node.go` — `ApplyTemplate`

The pattern `{{input.<fieldName>}}` is replaced with the string representation of `input[fieldName]`. Rules:

- Only top-level keys are supported (`{{input.foo}}`, not `{{input.foo.bar}}`).
- If the key does not exist in input, it is replaced with an empty string.
- Non-string values are formatted with `fmt.Sprintf("%v", val)`.
- Available in: `HttpNode` (URL, body, header values) and `LLMNode` (prompt, system_prompt).

---

## Built-in Nodes

### `echo`

Passes input through unchanged. No params.

**Use case:** debugging, no-op placeholder.

| | |
|---|---|
| **Params** | none |
| **Input** | any map |
| **Output** | identical to input |
| **Errors** | never fails |

```json
{ "id": "passthrough", "type": "echo" }
```

---

### `fail`

Always returns an error. No params.

**Use case:** testing error handling in pipelines.

| | |
|---|---|
| **Params** | none |
| **Input** | ignored |
| **Output** | none |
| **Errors** | always: `"this node always fails"` |

```json
{ "id": "intentional-failure", "type": "fail" }
```

---

### `math`

Adds two numbers and returns the sum. Values can come from params (load-time defaults) or from the input map (runtime override). Input takes precedence.

| Param | Type | Required | Description |
|---|---|---|---|
| `a` | `float64` | one of `a`/`b` must be present at runtime | First operand |
| `b` | `float64` | one of `a`/`b` must be present at runtime | Second operand |

**Input resolution order:** `input["a"]` → `params["a"]` → error.

**Output:**

```json
{ "result": 3.0 }
```

**Errors:**
- `"missing or invalid 'a' parameter"` — if neither input nor params has a valid `float64` for `a`
- `"missing or invalid 'b' parameter"` — same for `b`

**Example:**

```json
{
  "id": "add",
  "type": "math",
  "params": { "b": 10 }
}
```

With input `{"a": 5}` → output `{"result": 15}`.

---

### `http`

Makes an outbound HTTP request. Supports `{{input.field}}` template substitution in the URL, body, and header values.

**Source:** `internal/core/http_node.go`

| Param | Type | Default | Description |
|---|---|---|---|
| `url` | `string` | — | **Required.** Target URL. Supports `{{input.field}}` substitution. |
| `method` | `string` | `"GET"` | HTTP method: `GET`, `POST`, `PUT`, `PATCH`, `DELETE` |
| `body` | `string` | `""` | Request body. Supports `{{input.field}}` substitution. |
| `headers` | `map[string]string` | `{}` | Key-value headers. Values support substitution. |
| `timeout_seconds` | `float64` | `30` | Request timeout in seconds. |

**Output:**

```json
{
  "status_code": 200,
  "body": "...",
  "headers": {
    "Content-Type": "application/json"
  }
}
```

> Non-2xx responses do **not** cause an error — they return normally with the actual `status_code`. Validate status downstream with a `safe_object` node or in a subsequent node if needed.

**Errors:**
- `"missing required param: url"` — if `url` resolves to an empty string after substitution
- `"failed to create request: ..."` — invalid URL or method
- `"request failed: ..."` — network error, timeout, DNS failure
- `"failed to read response body: ..."` — I/O error reading the response

**Example — POST with dynamic body:**

```json
{
  "id": "notify-slack",
  "type": "http",
  "params": {
    "url": "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
    "method": "POST",
    "headers": { "Content-Type": "application/json" },
    "body": "{\"text\": \"Result: {{input.response}}\"}"
  }
}
```

---

### `llm`

Calls the OpenAI Chat Completions API. Supports `{{input.field}}` substitution in both `prompt` and `system_prompt`.

**Source:** `internal/core/llm_node.go`

| Param | Type | Default | Description |
|---|---|---|---|
| `prompt` | `string` | — | **Required** (unless `input["prompt"]` is set at runtime). The user message. Supports substitution. |
| `model` | `string` | `"gpt-4o-mini"` | OpenAI model ID. |
| `system_prompt` | `string` | `""` | System message prepended to the conversation. Supports substitution. |
| `max_tokens` | `float64` | `0` (API default) | Maximum tokens in the completion. |
| `temperature` | `float64` | `0` (API default) | Sampling temperature (0.0–2.0). |
| `api_key` | `string` | `""` | OpenAI API key. If empty, the request will fail with a 401. Prefer injecting via environment variable handling upstream. |

**Runtime override:** if `input["prompt"]` is a non-empty string, it overrides the `prompt` param entirely (no substitution applied to the runtime value).

**Output:**

```json
{
  "response": "The completion text from the model.",
  "model": "gpt-4o-mini",
  "prompt_tokens": 42,
  "completion_tokens": 128,
  "total_tokens": 170
}
```

**Errors:**
- `"missing required field: prompt"` — no prompt from either input or params
- `"OpenAI API error (status 401): ..."` — bad or missing API key
- `"OpenAI API error (status 429): ..."` — rate limit
- `"no choices returned from API"` — unexpected empty response
- `"API request failed: ..."` — network error

**Example — summarization step:**

```json
{
  "id": "summarize",
  "type": "llm",
  "params": {
    "prompt": "Summarize the following in one sentence: {{input.body}}",
    "model": "gpt-4o-mini",
    "system_prompt": "You are a concise summarizer. Return only the summary.",
    "max_tokens": 100,
    "api_key": "sk-..."
  }
}
```

---

### `safe_object`

Validates the input map against a declared schema. Fields not listed in the schema are stripped from the output (allow-list). Fields in the schema that are missing or have the wrong type cause an error.

**Source:** `internal/core/safe_object_node.go`

| Param | Type | Description |
|---|---|---|
| `schema` | `map[string]string` | Map of field name → expected type. Supported types: `"string"`, `"float64"`. |

**Output:** a new map containing only the fields declared in the schema, with their original values.

**Errors:**
- `"missing required field: <key>"` — a schema key is absent from input
- `"invalid type for field <key>: expected string"` — type mismatch
- `"invalid type for field <key>: expected float64"` — type mismatch
- `"unsupported schema type: <type>"` — schema uses a type other than `"string"` or `"float64"`

> **Note:** `safe_object` is registered in the engine's node registry but is not listed in the frontend's `NODE_TYPES` (`web/src/lib/nodes.ts`). It can be used via direct API calls but will not appear in the visual builder.

**Example — validate LLM output before downstream use:**

```json
{
  "id": "validate-output",
  "type": "safe_object",
  "params": {
    "schema": {
      "response": "string",
      "total_tokens": "float64"
    }
  }
}
```

With `llm` output as input → strips `model`, `prompt_tokens`, `completion_tokens`, and passes only `response` and `total_tokens` forward.

---

## Node I/O Summary

| Node | Key input fields | Key output fields |
|---|---|---|
| `echo` | any | same as input |
| `fail` | — | — (always errors) |
| `math` | `a` (float64), `b` (float64) | `result` (float64) |
| `http` | any (for template substitution) | `status_code`, `body`, `headers` |
| `llm` | `prompt` (optional override) | `response`, `model`, `prompt_tokens`, `completion_tokens`, `total_tokens` |
| `safe_object` | fields matching schema | only schema-declared fields |
