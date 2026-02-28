package core

import (
	"context"
	"fmt"
)

// LLMNode is a stub for an Agentic AI Node that integrates with LLM APIs.
type LLMNode struct {
	Prompt *string
}

// Execute implements the NodeExecutor interface for LLMNode.
func (n LLMNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	var prompt string
	if val, ok := input["prompt"].(string); ok && val != "" {
		prompt = val
	} else if n.Prompt != nil && *n.Prompt != "" {
		prompt = *n.Prompt
	} else {
		return nil, fmt.Errorf("missing required field: prompt")
	}

	return map[string]any{
		"response": "mock LLM response for: " + prompt,
	}, nil
}
