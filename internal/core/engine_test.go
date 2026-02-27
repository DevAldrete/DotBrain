package core_test

import (
	"context"
	"testing"

	"github.com/devaldrete/dotbrain/internal/core"
)

func TestEngine_SequentialExecution(t *testing.T) {
	engine := core.NewEngine()

	engine.Register(core.EchoNode{})
	engine.Register(core.MathNode{})

	ctx := context.Background()
	input := map[string]any{
		"a": 5.0,
		"b": 10.0,
	}

	result, err := engine.Execute(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := result["result"].(float64)
	if !ok {
		t.Fatalf("expected result of type float64")
	}

	if val != 15.0 {
		t.Errorf("expected 15.0, got %v", val)
	}
}

func TestEngine_NodeFailure(t *testing.T) {
	engine := core.NewEngine()

	engine.Register(core.FailNode{})
	engine.Register(core.MathNode{})

	ctx := context.Background()
	input := map[string]any{
		"a": 5.0,
		"b": 10.0,
	}

	_, err := engine.Execute(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Error() != "node execution failed: this node always fails" {
		t.Errorf("unexpected error message: %v", err)
	}
}
