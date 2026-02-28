# TASK-05 — Implement `HttpNode`

**Phase:** 3 — New Node Types  
**Priority:** Medium  
**Depends on:** TASK-01 (param injection — HttpNode is parameterized)  
**Files affected:** `internal/core/http_node.go` (new file), `internal/core/engine.go`, `internal/core/http_node_test.go` (new file)

---

## Problem

The "general automation" use case requires nodes that can call external HTTP APIs. There is no such node type today. Without `HttpNode`, workflows are limited to math operations and LLM calls, which makes the system impractical for real automation pipelines.

---

## Goal

Implement an `HttpNode` that makes an outbound HTTP request and returns the response body. Register it in the engine's node registry so it can be used in workflow definitions.

---

## Node Configuration

Params (set in the workflow definition, via `NodeConfig.Params`):

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | The URL to call. Supports `{{input.field}}` template substitution. |
| `method` | string | No | HTTP method. Default: `GET`. |
| `headers` | map[string]string | No | Additional request headers. |
| `body` | string | No | Request body template. Supports `{{input.field}}` substitution. |
| `timeout_seconds` | number | No | Request timeout. Default: `30`. Max: `300`. |

Output map (on success):

| Key | Type | Description |
|-----|------|-------------|
| `status_code` | int | HTTP response status code |
| `body` | string | Raw response body |
| `headers` | map[string]string | Response headers |

---

## Template Substitution

`{{input.field}}` references are replaced with the corresponding value from the input map before the request is made. This allows the URL and body to be dynamically constructed from previous nodes' outputs.

Example:
```json
{
  "id": "call-api",
  "type": "http",
  "params": {
    "url": "https://api.example.com/users/{{input.user_id}}",
    "method": "POST",
    "body": "{\"message\": \"{{input.message}}\"}"
  }
}
```

---

## Acceptance Criteria

- [ ] `HttpNode` struct exists in `internal/core/http_node.go`
- [ ] `HttpNode` implements `NodeExecutor`
- [ ] `HttpNode` is initialized with `map[string]any` params (matching TASK-01 factory signature)
- [ ] Missing `url` param returns an error from `Execute`
- [ ] Successful HTTP request returns `status_code`, `body`, and `headers` in the output map
- [ ] Non-2xx responses do NOT automatically return an error (the workflow decides what to do with the status code); the response is passed through regardless
- [ ] `{{input.field}}` substitution works in `url` and `body`
- [ ] A configurable timeout is respected; exceeding it returns an error
- [ ] `HttpNode` is registered in `nodeRegistry` under the key `"http"`
- [ ] `go test ./internal/core/...` passes with full test coverage for the node

---

## TDD Approach

### Red — write failing tests first

**File:** `internal/core/http_node_test.go`

Use `net/http/httptest.NewServer` to create a local test HTTP server — no real network calls in tests.

```go
// TestHttpNode_Execute_MissingURL verifies that Execute returns an error
// when the "url" param is absent.
func TestHttpNode_Execute_MissingURL(t *testing.T) {
    node := HttpNode{Params: map[string]any{}}
    _, err := node.Execute(context.Background(), map[string]any{})
    if err == nil {
        t.Fatal("expected error for missing url param")
    }
}

// TestHttpNode_Execute_GET verifies a successful GET request returns
// the correct status code and body.
func TestHttpNode_Execute_GET(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        w.Write([]byte(`{"ok":true}`))
    }))
    defer srv.Close()

    node := HttpNode{Params: map[string]any{"url": srv.URL, "method": "GET"}}
    out, err := node.Execute(context.Background(), map[string]any{})
    if err != nil {
        t.Fatal(err)
    }
    if out["status_code"].(int) != 200 {
        t.Errorf("expected 200, got %v", out["status_code"])
    }
}

// TestHttpNode_Execute_TemplateSubstitution verifies that {{input.id}}
// is replaced with the value from the input map.
func TestHttpNode_Execute_TemplateSubstitution(t *testing.T) { ... }

// TestHttpNode_Execute_Timeout verifies that a request that exceeds
// timeout_seconds returns an error.
func TestHttpNode_Execute_Timeout(t *testing.T) { ... }
```

### Green — minimal implementation

```go
type HttpNode struct {
    Params map[string]any
}

func (h HttpNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
    rawURL, ok := h.Params["url"].(string)
    if !ok || rawURL == "" {
        return nil, fmt.Errorf("http node: missing required param 'url'")
    }

    // Apply template substitution
    finalURL := applyTemplate(rawURL, input)

    // Build HTTP client with timeout
    timeoutSec := 30
    if t, ok := h.Params["timeout_seconds"].(float64); ok {
        timeoutSec = int(t)
    }
    client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}

    // Build request
    method := "GET"
    if m, ok := h.Params["method"].(string); ok && m != "" {
        method = strings.ToUpper(m)
    }

    var bodyReader io.Reader
    if bodyTmpl, ok := h.Params["body"].(string); ok && bodyTmpl != "" {
        bodyReader = strings.NewReader(applyTemplate(bodyTmpl, input))
    }

    req, err := http.NewRequestWithContext(ctx, method, finalURL, bodyReader)
    if err != nil {
        return nil, fmt.Errorf("http node: failed to create request: %w", err)
    }

    // Set headers
    if headers, ok := h.Params["headers"].(map[string]any); ok {
        for k, v := range headers {
            req.Header.Set(k, fmt.Sprintf("%v", v))
        }
    }

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("http node: request failed: %w", err)
    }
    defer resp.Body.Close()

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("http node: failed to read response body: %w", err)
    }

    respHeaders := map[string]string{}
    for k, vals := range resp.Header {
        respHeaders[k] = strings.Join(vals, ", ")
    }

    return map[string]any{
        "status_code": resp.StatusCode,
        "body":        string(respBody),
        "headers":     respHeaders,
    }, nil
}
```

**`applyTemplate` helper:**
```go
// applyTemplate replaces {{input.key}} with values from the input map.
func applyTemplate(tmpl string, input map[string]any) string {
    for k, v := range input {
        tmpl = strings.ReplaceAll(tmpl, "{{input."+k+"}}", fmt.Sprintf("%v", v))
    }
    return tmpl
}
```

### Refactor

- Move `applyTemplate` to a shared `internal/core/template.go` file since `LLMNode` (TASK-06) will need the same substitution logic.
- Consider whether a non-2xx response should set an `"error"` key in the output map as a convenience (but still not return an error from `Execute`).

---

## Definition of Done

- `go test ./internal/core/...` passes with `HttpNode` tests
- `go test ./...` (full suite) passes — no regressions
- The node is registered as `"http"` in the engine registry
- A workflow using `"type": "http"` can be triggered and completes successfully
