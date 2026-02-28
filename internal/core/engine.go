package core

import (
	"context"
	"fmt"
)

var nodeRegistry = map[string]func() NodeExecutor{
	"echo": func() NodeExecutor { return EchoNode{} },
	"fail": func() NodeExecutor { return FailNode{} },
	"math": func() NodeExecutor { return MathNode{} },
	// Other nodes can be registered here or via a registration function
}

// Engine is a simple orchestrator that executes a slice of NodeExecutors sequentially.
type Engine struct {
	nodes []NodeExecutor
}

// NewEngine creates a new Engine instance.
func NewEngine() *Engine {
	return &Engine{
		nodes: make([]NodeExecutor, 0),
	}
}

// LoadFromDefinition instantiates nodes from a workflow definition and registers them.
func (e *Engine) LoadFromDefinition(def *WorkflowDefinition) error {
	for _, config := range def.Nodes {
		factory, exists := nodeRegistry[config.Type]
		if !exists {
			return fmt.Errorf("unknown node type: %s", config.Type)
		}
		// For a more advanced setup, we would initialize the node with config.Params here
		e.Register(factory())
	}
	return nil
}

// Register adds a NodeExecutor to the engine's execution sequence.
func (e *Engine) Register(node NodeExecutor) {
	e.nodes = append(e.nodes, node)
}

// Execute runs the registered nodes sequentially, passing the output of one
// as the input to the next.
func (e *Engine) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	currentData := input

	for _, node := range e.nodes {
		output, err := node.Execute(ctx, currentData)
		if err != nil {
			return nil, fmt.Errorf("node execution failed: %w", err)
		}

		// Optional: We can merge outputs or replace them. The simple requirement
		// says "passing output of Node A as input to Node B". Let's replace for now,
		// but typically we'd merge if the orchestrator is passing the accumulated state.
		// For a simple sequence where node B only relies on node A's output, replacement works.
		currentData = output
	}

	return currentData, nil
}
