package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// LLMNode calls the OpenAI Chat Completions API.
type LLMNode struct {
	Prompt       *string
	Model        string
	SystemPrompt string
	MaxTokens    int
	Temperature  float64
	APIKey       string
	baseURL      string
	client       *http.Client
}

// NewLLMNode creates an LLMNode from params.
func NewLLMNode(params map[string]any) *LLMNode {
	node := &LLMNode{
		Model:   "gpt-4o-mini",
		baseURL: "https://api.openai.com",
		client:  http.DefaultClient,
	}

	if prompt, ok := params["prompt"].(string); ok {
		node.Prompt = &prompt
	}
	if model, ok := params["model"].(string); ok {
		node.Model = model
	}
	if sp, ok := params["system_prompt"].(string); ok {
		node.SystemPrompt = sp
	}
	if mt, ok := params["max_tokens"].(float64); ok {
		node.MaxTokens = int(mt)
	}
	if temp, ok := params["temperature"].(float64); ok {
		node.Temperature = temp
	}
	if key, ok := params["api_key"].(string); ok {
		node.APIKey = key
	}

	return node
}

// SetBaseURL overrides the API base URL (used for testing).
func (n *LLMNode) SetBaseURL(url string) {
	n.baseURL = url
}

// Execute calls the OpenAI Chat Completions API.
func (n *LLMNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	// Resolve prompt: input overrides params
	var prompt string
	if val, ok := input["prompt"].(string); ok && val != "" {
		prompt = val
	} else if n.Prompt != nil && *n.Prompt != "" {
		prompt = ApplyTemplate(*n.Prompt, input)
	} else {
		return nil, fmt.Errorf("missing required field: prompt")
	}

	// Build messages
	var messages []map[string]string
	if n.SystemPrompt != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": ApplyTemplate(n.SystemPrompt, input),
		})
	}
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": prompt,
	})

	// Build request body
	reqBody := map[string]any{
		"model":    n.Model,
		"messages": messages,
	}
	if n.MaxTokens > 0 {
		reqBody["max_tokens"] = n.MaxTokens
	}
	if n.Temperature > 0 {
		reqBody["temperature"] = n.Temperature
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	url := n.baseURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+n.APIKey)

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	// Parse response
	var respBody struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Model string `json:"model"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBytes, &respBody); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(respBody.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from API")
	}

	return map[string]any{
		"response":          respBody.Choices[0].Message.Content,
		"model":             respBody.Model,
		"prompt_tokens":     respBody.Usage.PromptTokens,
		"completion_tokens": respBody.Usage.CompletionTokens,
		"total_tokens":      respBody.Usage.TotalTokens,
	}, nil
}
