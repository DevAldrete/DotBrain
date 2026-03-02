package core_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devaldrete/dotbrain/internal/core"
)

// mockOpenAIServer creates a test server that mimics the OpenAI Chat Completions API.
func mockOpenAIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func TestLLMNode_Execute_MissingPrompt(t *testing.T) {
	node := core.NewLLMNode(map[string]any{})

	_, err := node.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing prompt, got nil")
	}

	expectedErr := "missing required field: prompt"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestLLMNode_Execute_ValidPrompt(t *testing.T) {
	server := mockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("expected /v1/chat/completions, got %s", r.URL.Path)
		}

		// Verify auth header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %q", auth)
		}

		// Parse request body
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		messages := reqBody["messages"].([]any)
		lastMsg := messages[len(messages)-1].(map[string]any)
		if lastMsg["content"] != "Hello world" {
			t.Errorf("expected prompt 'Hello world', got %q", lastMsg["content"])
		}

		// Return mock response
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "Hi there!",
					},
				},
			},
			"model": "gpt-4o-mini",
			"usage": map[string]any{
				"prompt_tokens":     10.0,
				"completion_tokens": 5.0,
				"total_tokens":      15.0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	node := core.NewLLMNode(map[string]any{
		"model":   "gpt-4o-mini",
		"api_key": "test-key",
	})
	node.SetBaseURL(server.URL)

	result, err := node.Execute(context.Background(), map[string]any{
		"prompt": "Hello world",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	response, ok := result["response"].(string)
	if !ok {
		t.Fatal("expected 'response' field of type string")
	}
	if response != "Hi there!" {
		t.Errorf("expected 'Hi there!', got %q", response)
	}

	model, ok := result["model"].(string)
	if !ok {
		t.Fatal("expected 'model' field of type string")
	}
	if model != "gpt-4o-mini" {
		t.Errorf("expected 'gpt-4o-mini', got %q", model)
	}

	promptTokens, ok := result["prompt_tokens"].(int)
	if !ok {
		t.Fatalf("expected 'prompt_tokens' of type int, got %T", result["prompt_tokens"])
	}
	if promptTokens != 10 {
		t.Errorf("expected 10 prompt tokens, got %d", promptTokens)
	}

	totalTokens := result["total_tokens"].(int)
	if totalTokens != 15 {
		t.Errorf("expected 15 total tokens, got %d", totalTokens)
	}
}

func TestLLMNode_Execute_WithSystemPrompt(t *testing.T) {
	server := mockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		messages := reqBody["messages"].([]any)
		if len(messages) != 2 {
			t.Errorf("expected 2 messages (system + user), got %d", len(messages))
		}

		systemMsg := messages[0].(map[string]any)
		if systemMsg["role"] != "system" {
			t.Errorf("expected first message role 'system', got %q", systemMsg["role"])
		}
		if systemMsg["content"] != "You are a helpful assistant." {
			t.Errorf("unexpected system prompt: %q", systemMsg["content"])
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "response"}},
			},
			"model": "gpt-4o-mini",
			"usage": map[string]any{
				"prompt_tokens":     5.0,
				"completion_tokens": 3.0,
				"total_tokens":      8.0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	node := core.NewLLMNode(map[string]any{
		"system_prompt": "You are a helpful assistant.",
		"api_key":       "test-key",
	})
	node.SetBaseURL(server.URL)

	_, err := node.Execute(context.Background(), map[string]any{
		"prompt": "Hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLLMNode_Execute_TemplateSubstitution(t *testing.T) {
	server := mockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		messages := reqBody["messages"].([]any)
		lastMsg := messages[len(messages)-1].(map[string]any)
		if lastMsg["content"] != "Summarize: This is the text to summarize" {
			t.Errorf("expected template-substituted prompt, got %q", lastMsg["content"])
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "summary"}},
			},
			"model": "gpt-4o-mini",
			"usage": map[string]any{
				"prompt_tokens":     5.0,
				"completion_tokens": 3.0,
				"total_tokens":      8.0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	node := core.NewLLMNode(map[string]any{
		"prompt":  "Summarize: {{input.text}}",
		"api_key": "test-key",
	})
	node.SetBaseURL(server.URL)

	_, err := node.Execute(context.Background(), map[string]any{
		"text": "This is the text to summarize",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLLMNode_Execute_WithParams(t *testing.T) {
	server := mockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		messages := reqBody["messages"].([]any)
		lastMsg := messages[len(messages)-1].(map[string]any)

		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "response for: " + lastMsg["content"].(string)}},
			},
			"model": "gpt-4o-mini",
			"usage": map[string]any{
				"prompt_tokens":     5.0,
				"completion_tokens": 3.0,
				"total_tokens":      8.0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	node := core.NewLLMNode(map[string]any{
		"prompt":  "Default prompt from params",
		"api_key": "test-key",
	})
	node.SetBaseURL(server.URL)

	// Should use param prompt when input is empty
	result, err := node.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	response := result["response"].(string)
	if response != "response for: Default prompt from params" {
		t.Errorf("expected param prompt to be used, got %q", response)
	}

	// Input prompt should override param prompt
	result2, err := node.Execute(context.Background(), map[string]any{
		"prompt": "Input prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	response2 := result2["response"].(string)
	if response2 != "response for: Input prompt" {
		t.Errorf("expected input prompt to override, got %q", response2)
	}
}

func TestLLMNode_Execute_APIError(t *testing.T) {
	server := mockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "rate limit exceeded",
			},
		})
	})
	defer server.Close()

	node := core.NewLLMNode(map[string]any{
		"api_key": "test-key",
	})
	node.SetBaseURL(server.URL)

	_, err := node.Execute(context.Background(), map[string]any{
		"prompt": "Hello",
	})
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}
