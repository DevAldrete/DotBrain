# TASK-16 — Multi-Provider LLM Node

**Phase:** 9 — Node Library  
**Priority:** Medium  
**Depends on:** nothing (isolated to `llm_node.go`)  
**Files affected:** `internal/core/llm_node.go`, `internal/core/llm_node_test.go`, `web/src/lib/nodes.ts`, `web/src/lib/types.ts`

---

## Problem

`LLMNode` is hardcoded to `https://api.openai.com`. The README promises "a unified interface for external LLM APIs (OpenAI, Anthropic, etc.)" but only OpenAI is implemented.

Additionally, the API key is passed as a plain node param (`"api_key": "sk-..."`), which means it is stored in plaintext in the `workflows.definition` JSONB column and in `node_executions.input_data`. This is a security problem.

---

## Goal

1. Add a `provider` param to `LLMNode` supporting `"openai"`, `"anthropic"`, and `"ollama"` (local models).
2. Read API keys from environment variables instead of node params, so keys are never persisted to the database.
3. Keep the existing output shape identical so downstream nodes are unaffected.

---

## Provider Interface

```go
// internal/core/llm_provider.go

type LLMRequest struct {
    Model        string
    SystemPrompt string
    UserPrompt   string
    MaxTokens    int
    Temperature  float64
}

type LLMResponse struct {
    Content          string
    Model            string
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}

type LLMProvider interface {
    Complete(ctx context.Context, req LLMRequest) (LLMResponse, error)
}
```

### Providers

**OpenAI** (`internal/core/llm_openai.go`):
- Endpoint: `https://api.openai.com/v1/chat/completions`
- Auth: `Authorization: Bearer $OPENAI_API_KEY`
- Request/response: current `LLMNode` implementation, extracted

**Anthropic** (`internal/core/llm_anthropic.go`):
- Endpoint: `https://api.anthropic.com/v1/messages`
- Auth: `x-api-key: $ANTHROPIC_API_KEY`
- Request format: Anthropic Messages API (different from OpenAI)
- Response: map to `LLMResponse`

**Ollama** (`internal/core/llm_ollama.go`):
- Endpoint: `http://localhost:11434/api/chat` (configurable via `OLLAMA_BASE_URL`)
- Auth: none
- Request format: Ollama chat API
- Use case: local development and self-hosted models

---

## Updated `LLMNode`

```go
// internal/core/llm_node.go

type LLMNode struct {
    Provider     string   // "openai" | "anthropic" | "ollama"
    Model        string
    Prompt       *string
    SystemPrompt string
    MaxTokens    int
    Temperature  float64
    // api_key param is REMOVED — read from env inside Execute
}

func NewLLMNode(params map[string]any) *LLMNode {
    node := &LLMNode{
        Provider: "openai",
        Model:    "gpt-4o-mini",
    }
    if p, ok := params["provider"].(string); ok { node.Provider = p }
    if m, ok := params["model"].(string); ok { node.Model = m }
    // ... other params
    // NOTE: no api_key param — intentionally removed
    return node
}

func (n *LLMNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
    provider, err := n.buildProvider()
    if err != nil {
        return nil, err
    }
    // ... resolve prompt, build LLMRequest, call provider.Complete, return output map
}

func (n *LLMNode) buildProvider() (LLMProvider, error) {
    switch n.Provider {
    case "openai", "":
        key := os.Getenv("OPENAI_API_KEY")
        if key == "" {
            return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
        }
        return NewOpenAIProvider(key, http.DefaultClient), nil
    case "anthropic":
        key := os.Getenv("ANTHROPIC_API_KEY")
        if key == "" {
            return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is not set")
        }
        return NewAnthropicProvider(key, http.DefaultClient), nil
    case "ollama":
        baseURL := os.Getenv("OLLAMA_BASE_URL")
        if baseURL == "" {
            baseURL = "http://localhost:11434"
        }
        return NewOllamaProvider(baseURL, http.DefaultClient), nil
    default:
        return nil, fmt.Errorf("unknown LLM provider: %s", n.Provider)
    }
}
```

---

## Migration: Existing Workflows with `api_key` Param

Existing workflow definitions that include `"api_key": "sk-..."` in the `llm` node params will continue to parse without error (extra params are ignored by `NewLLMNode`). The key will not be used; the key is now read from the environment. This is a **silent breaking change** for anyone relying on per-node keys.

Document this clearly in the release notes.

---

## Updated Node Metadata (Frontend)

```ts
// web/src/lib/nodes.ts
{
    type: 'llm',
    params: [
        {
            key: 'provider',
            label: 'Provider',
            type: 'select',
            default: 'openai',
            options: [
                { value: 'openai', label: 'OpenAI' },
                { value: 'anthropic', label: 'Anthropic' },
                { value: 'ollama', label: 'Ollama (local)' }
            ]
        },
        {
            key: 'model',
            label: 'Model',
            type: 'string',
            placeholder: 'gpt-4o-mini / claude-3-5-sonnet-20241022 / llama3.2'
        },
        // prompt, system_prompt, max_tokens, temperature — unchanged
        // api_key param REMOVED
    ]
}
```

---

## Acceptance Criteria

- [ ] `LLMNode` accepts a `provider` param: `"openai"`, `"anthropic"`, `"ollama"`
- [ ] API keys are read from environment variables; `api_key` param is no longer accepted
- [ ] OpenAI provider works identically to the current implementation
- [ ] Anthropic provider calls the Messages API and maps the response to the shared output shape
- [ ] Ollama provider calls the local API; `OLLAMA_BASE_URL` overrides the default
- [ ] Unknown provider returns an error at execute time
- [ ] Missing API key env var returns a clear error at execute time (not a panic)
- [ ] Output shape is unchanged: `response`, `model`, `prompt_tokens`, `completion_tokens`, `total_tokens`
- [ ] `go test ./internal/core/...` passes with tests for each provider using mock HTTP servers

---

## TDD Approach

```go
// TestLLMNode_OpenAI_UsesEnvKey — verify OPENAI_API_KEY is read from env
func TestLLMNode_OpenAI_UsesEnvKey(t *testing.T) { ... }

// TestLLMNode_Anthropic_Execute — mock Anthropic API; verify correct request format
func TestLLMNode_Anthropic_Execute(t *testing.T) { ... }

// TestLLMNode_Ollama_Execute — mock Ollama API
func TestLLMNode_Ollama_Execute(t *testing.T) { ... }

// TestLLMNode_UnknownProvider — assert error
func TestLLMNode_UnknownProvider(t *testing.T) { ... }

// TestLLMNode_MissingAPIKey — assert error when env var absent
func TestLLMNode_MissingAPIKey(t *testing.T) { ... }
```

---

## Definition of Done

- All acceptance criteria checked
- `go test ./...` passes with no regressions
- `.env.example` updated: documents `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `OLLAMA_BASE_URL`
- `docs/core/nodes.md` updated for `llm`: remove `api_key` param, add `provider` param, document env vars
