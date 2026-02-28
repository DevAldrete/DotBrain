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

// TestEngine_LoadFromDefinition_PassesParamsToNode verifies that params
// defined in a NodeConfig are passed to the node at instantiation time.
func TestEngine_LoadFromDefinition_PassesParamsToNode(t *testing.T) {
	def := &core.WorkflowDefinition{
		Nodes: []core.NodeConfig{
			{
				ID:   "1",
				Type: "math",
				Params: map[string]any{
					"a": 10.0,
					"b": 20.0,
				},
			},
		},
	}

	engine := core.NewEngine()
	err := engine.LoadFromDefinition(def)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Execute with empty input to see if it uses the params
	result, err := engine.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := result["result"].(float64)
	if !ok || val != 30.0 {
		t.Errorf("expected 30.0, got %v", val)
	}
}

// TestEngine_LoadFromDefinition_NilParamsSafe verifies that a NodeConfig
// with no params field does not panic (passes an empty map instead of nil).
func TestEngine_LoadFromDefinition_NilParamsSafe(t *testing.T) {
	def := &core.WorkflowDefinition{
		Nodes: []core.NodeConfig{
			{ID: "1", Type: "echo"}, // no Params field
		},
	}
	engine := core.NewEngine()
	err := engine.LoadFromDefinition(def)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should execute echo node successfully
	result, err := engine.Execute(context.Background(), map[string]any{"hello": "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["hello"] != "world" {
		t.Errorf("expected world, got %v", result["hello"])
	}
}
