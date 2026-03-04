package core

import (
	"context"
	"fmt"
	"time"
)

// TimerNode introduces a delay (sleep) into the workflow execution.
// It respects context cancellation, so cancelled runs will interrupt the sleep.
//
// Params:
//   - duration_ms: delay in milliseconds (default: 1000)
//   - duration_s: delay in seconds (overrides duration_ms if set)
//
// Output: passes through input with added timing metadata.
type TimerNode struct {
	DurationMs int
}

// NewTimerNode creates a TimerNode from params.
func NewTimerNode(params map[string]any) *TimerNode {
	node := &TimerNode{
		DurationMs: 1000, // default 1 second
	}

	// Check seconds first (higher priority)
	if secs, ok := params["duration_s"].(float64); ok {
		node.DurationMs = int(secs * 1000)
	} else if ms, ok := params["duration_ms"].(float64); ok {
		node.DurationMs = int(ms)
	}

	// Clamp to reasonable range
	if node.DurationMs < 0 {
		node.DurationMs = 0
	}
	if node.DurationMs > 3600000 { // max 1 hour
		node.DurationMs = 3600000
	}

	return node
}

// Execute delays for the configured duration, then passes through the input
// with timing metadata. Respects context cancellation.
func (n *TimerNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	if n.DurationMs <= 0 {
		// No delay needed, just pass through
		output := make(map[string]any)
		for k, v := range input {
			output[k] = v
		}
		output["timer_waited_ms"] = float64(0)
		return output, nil
	}

	start := time.Now()
	duration := time.Duration(n.DurationMs) * time.Millisecond

	select {
	case <-time.After(duration):
		// Timer completed normally
		elapsed := time.Since(start)
		output := make(map[string]any)
		for k, v := range input {
			output[k] = v
		}
		output["timer_waited_ms"] = float64(elapsed.Milliseconds())
		return output, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("timer interrupted: %w", ctx.Err())
	}
}
