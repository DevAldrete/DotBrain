package core_test

import (
	"context"
	"testing"

	"github.com/devaldrete/dotbrain/internal/core"
)

func TestLLMNode_Execute_MissingPrompt(t *testing.T) {
	node := core.LLMNode{}
	ctx := context.Background()

	input := map[string]any{} // missing prompt

	_, err := node.Execute(ctx, input)
	if err == nil {
		t.Fatal("expected error for missing prompt, got nil")
	}

	expectedErr := "missing required field: prompt"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestLLMNode_Execute_ValidPrompt(t *testing.T) {
	node := core.LLMNode{}
	ctx := context.Background()

	input := map[string]any{
		"prompt": "Hello world",
	}

	result, err := node.Execute(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	response, ok := result["response"].(string)
	if !ok {
		t.Fatal("expected 'response' field of type string in result")
	}

	if response == "" {
		t.Error("expected non-empty response")
	}
}

func TestLLMNode_Execute_WithParams(t *testing.T) {
	prompt := "Default prompt from params"
	node := core.LLMNode{Prompt: &prompt}
	ctx := context.Background()

	// Should use prompt from params when missing in input
	input := map[string]any{}

	result, err := node.Execute(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	response, ok := result["response"].(string)
	if !ok {
		t.Fatal("expected 'response' field of type string in result")
	}

	expectedPrefix := "mock LLM response for: Default prompt from params"
	if response != expectedPrefix {
		t.Errorf("expected response to be %q, got %q", expectedPrefix, response)
	}

	// Should be overridden by input
	inputWithPrompt := map[string]any{
		"prompt": "Input prompt",
	}

	result2, err := node.Execute(ctx, inputWithPrompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	response2 := result2["response"].(string)
	expectedPrefix2 := "mock LLM response for: Input prompt"
	if response2 != expectedPrefix2 {
		t.Errorf("expected response to be %q, got %q", expectedPrefix2, response2)
	}
}
