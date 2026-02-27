package core

import (
	"context"
	"fmt"
)

type NodeExecutor interface {
	Execute(ctx context.Context, input map[string]any) (map[string]any, error)
}

type EchoNode struct{}

func (e EchoNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	return input, nil
}

type FailNode struct{}

func (f FailNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	return nil, fmt.Errorf("this node always fails")
}

type MathNode struct{}

func (m MathNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	v1, ok := input["a"].(float64)

	if !ok {
		return nil, fmt.Errorf("missing or invalid 'a' parameter")
	}

	v2, ok := input["b"].(float64)

	if !ok {
		return nil, fmt.Errorf("missing or invalid 'b' parameter")
	}

	result := v1 + v2

	return map[string]any{
		"result": result,
	}, nil
}
