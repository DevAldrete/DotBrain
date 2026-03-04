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

// CountingFailNode fails the first FailTimes invocations and then succeeds
// by returning the input (like EchoNode). It is safe for concurrent use
// because each instance is only used by a single DAG node.
type CountingFailNode struct {
	FailTimes int
	calls     int
}

func (c *CountingFailNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	c.calls++
	if c.calls <= c.FailTimes {
		return nil, fmt.Errorf("counting_fail: attempt %d of %d", c.calls, c.FailTimes)
	}
	return input, nil
}

type MathNode struct {
	A *float64
	B *float64
}

func (m MathNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	var a float64
	if val, ok := input["a"].(float64); ok {
		a = val
	} else if m.A != nil {
		a = *m.A
	} else {
		return nil, fmt.Errorf("missing or invalid 'a' parameter")
	}

	var b float64
	if val, ok := input["b"].(float64); ok {
		b = val
	} else if m.B != nil {
		b = *m.B
	} else {
		return nil, fmt.Errorf("missing or invalid 'b' parameter")
	}

	result := a + b

	return map[string]any{
		"result": result,
	}, nil
}
