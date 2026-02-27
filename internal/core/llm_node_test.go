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
