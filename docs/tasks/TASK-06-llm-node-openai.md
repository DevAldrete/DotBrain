# TASK-06 — Implement `LLMNode` with Real OpenAI API

**Phase:** 3 — New Node Types  
**Priority:** Medium  
**Depends on:** TASK-01 (param injection), TASK-05 (for `applyTemplate` helper)  
**Files affected:** `internal/core/llm_node.go` (rewrite), `internal/core/engine.go`, `go.mod`

---

## Problem

`LLMNode` is currently a stub that returns a hardcoded mock string. It is not registered in the engine's node registry. `OPENAI_API_KEY` is defined in `.env.example` but is never read by any code.

The core value proposition of DotBrain includes AI-powered workflow nodes. Without a real LLM integration, the system cannot deliver on this.

---

## Goal

Rewrite `LLMNode` to call the OpenAI Chat Completions API. Register it in the engine registry. Allow the prompt and model to be configured via `NodeConfig.Params` with support for `{{input.field}}` template substitution.

---

## Node Configuration

Params (set in the workflow definition via `NodeConfig.Params`):

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `prompt` | string | Yes | User message. Supports `{{input.field}}` substitution. |
| `model` | string | No | OpenAI model ID. Default: `gpt-4o-mini`. |
| `system_prompt` | string | No | System message prepended to the conversation. |
| `max_tokens` | number | No | Maximum tokens in the response. Default: `1024`. |
| `temperature` | number | No | Sampling temperature (0.0–2.0). Default: `1.0`. |

Output map (on success):

| Key | Type | Description |
|-----|------|-------------|
| `response` | string | The text content of the first choice |
| `model` | string | The model that was actually used |
| `usage_prompt_tokens` | int | Prompt token count |
| `usage_completion_tokens` | int | Completion token count |
| `usage_total_tokens` | int | Total token count |

---

## API Key Handling

The `OPENAI_API_KEY` environment variable is read at startup and injected into the node at instantiation time via the factory:

```go
"llm": func(p map[string]any) NodeExecutor {
    return LLMNode{
        Params: p,
        APIKey: os.Getenv("OPENAI_API_KEY"),
    }
},
```

The node should return a clear error if `APIKey` is empty when `Execute` is called.

---

## Acceptance Criteria

- [ ] `LLMNode` is rewritten with real OpenAI API calls (not a stub)
- [ ] `LLMNode` is registered in `nodeRegistry` under the key `"llm"`
- [ ] Missing `prompt` param returns a descriptive error
- [ ] Empty `APIKey` returns a descriptive error (not a cryptic HTTP 401)
- [ ] `{{input.field}}` substitution is applied to the `prompt` before sending to OpenAI
- [ ] Output map contains `response`, `model`, and token usage fields
- [ ] API errors (rate limit, invalid key, etc.) are returned as errors from `Execute` with the OpenAI error message included
- [ ] Unit tests mock the HTTP transport — no real OpenAI API calls in tests
- [ ] `go test ./internal/core/...` passes

---

## Library Choice

Use `github.com/sashabaranov/go-openai` — the standard, well-maintained Go OpenAI client.

```bash
go get github.com/sashabaranov/go-openai
```

---

## TDD Approach

### Red — write failing tests first

**File:** `internal/core/llm_node_test.go`

The key constraint is that tests must not make real network calls. Use a custom `http.RoundTripper` to intercept and mock the OpenAI API response:

```go
// mockTransport intercepts HTTP calls and returns a preset response.
type mockTransport struct {
    response string
    status   int
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    body := io.NopCloser(strings.NewReader(m.response))
    return &http.Response{
        StatusCode: m.status,
        Body:       body,
        Header:     make(http.Header),
    }, nil
}

// TestLLMNode_Execute_MissingPrompt verifies that Execute returns an error
// when the "prompt" param is absent.
func TestLLMNode_Execute_MissingPrompt(t *testing.T) {
    node := LLMNode{Params: map[string]any{}, APIKey: "test-key"}
    _, err := node.Execute(context.Background(), map[string]any{})
    if err == nil {
        t.Fatal("expected error for missing prompt param")
    }
}

// TestLLMNode_Execute_EmptyAPIKey verifies that Execute returns a descriptive
// error when the API key is not set.
func TestLLMNode_Execute_EmptyAPIKey(t *testing.T) {
    node := LLMNode{Params: map[string]any{"prompt": "hello"}, APIKey: ""}
    _, err := node.Execute(context.Background(), map[string]any{})
    if err == nil {
        t.Fatal("expected error for empty API key")
    }
    if !strings.Contains(err.Error(), "API key") {
        t.Errorf("expected error to mention 'API key', got: %v", err)
    }
}

// TestLLMNode_Execute_Success verifies that a successful API response
// is returned correctly in the output map.
func TestLLMNode_Execute_Success(t *testing.T) {
    // Construct a valid OpenAI Chat Completions response JSON
    mockResp := `{
        "id": "chatcmpl-test",
        "object": "chat.completion",
        "model": "gpt-4o-mini",
        "choices": [{"message": {"role": "assistant", "content": "Hello!"}, "finish_reason": "stop", "index": 0}],
        "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
    }`

    // Inject mock transport into the go-openai client
    // (go-openai supports custom http.Client via ClientConfig)

    node := LLMNode{
        Params: map[string]any{"prompt": "Say hello"},
        APIKey: "test-key",
        httpClient: &http.Client{Transport: &mockTransport{response: mockResp, status: 200}},
    }
    out, err := node.Execute(context.Background(), map[string]any{})
    if err != nil {
        t.Fatal(err)
    }
    if out["response"] != "Hello!" {
        t.Errorf("expected 'Hello!', got %v", out["response"])
    }
}

// TestLLMNode_Execute_TemplateSubstitution verifies that {{input.topic}}
// is replaced in the prompt before sending.
func TestLLMNode_Execute_TemplateSubstitution(t *testing.T) { ... }
```

### Green — minimal implementation

```go
type LLMNode struct {
    Params     map[string]any
    APIKey     string
    httpClient *http.Client // injectable for testing; nil uses default
}

func (n LLMNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
    if n.APIKey == "" {
        return nil, fmt.Errorf("llm node: OPENAI_API_KEY is not set")
    }

    promptTmpl, ok := n.Params["prompt"].(string)
    if !ok || promptTmpl == "" {
        return nil, fmt.Errorf("llm node: missing required param 'prompt'")
    }
    prompt := applyTemplate(promptTmpl, input)

    model := "gpt-4o-mini"
    if m, ok := n.Params["model"].(string); ok && m != "" {
        model = m
    }

    cfg := openai.DefaultConfig(n.APIKey)
    if n.httpClient != nil {
        cfg.HTTPClient = n.httpClient
    }
    client := openai.NewClientWithConfig(cfg)

    messages := []openai.ChatCompletionMessage{}
    if sys, ok := n.Params["system_prompt"].(string); ok && sys != "" {
        messages = append(messages, openai.ChatCompletionMessage{
            Role:    openai.ChatMessageRoleSystem,
            Content: sys,
        })
    }
    messages = append(messages, openai.ChatCompletionMessage{
        Role:    openai.ChatMessageRoleUser,
        Content: prompt,
    })

    resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model:    model,
        Messages: messages,
    })
    if err != nil {
        return nil, fmt.Errorf("llm node: OpenAI API error: %w", err)
    }

    return map[string]any{
        "response":                 resp.Choices[0].Message.Content,
        "model":                    resp.Model,
        "usage_prompt_tokens":      resp.Usage.PromptTokens,
        "usage_completion_tokens":  resp.Usage.CompletionTokens,
        "usage_total_tokens":       resp.Usage.TotalTokens,
    }, nil
}
```

### Refactor

- Move `applyTemplate` (also used by `HttpNode`) to `internal/core/template.go` if not already done in TASK-05.
- Consider adding a `stream: true` param path later for streaming responses.

---

## Definition of Done

- `go test ./internal/core/...` passes with no real network calls
- `go test ./...` passes — no regressions
- The node is registered as `"llm"` in the engine registry
- With a valid `OPENAI_API_KEY` set, triggering a workflow with an `"llm"` node produces a real OpenAI response in the run's `output_data`
- The stub comment in `llm_node.go` is removed
