package core

import (
	"context"
	"fmt"
)

var nodeRegistry = map[string]func(map[string]any) NodeExecutor{
	"echo": func(p map[string]any) NodeExecutor { return EchoNode{} },
	"fail": func(p map[string]any) NodeExecutor { return FailNode{} },
	"math": func(p map[string]any) NodeExecutor {
		node := MathNode{}
		if val, ok := p["a"].(float64); ok {
			node.A = &val
		}
		if val, ok := p["b"].(float64); ok {
			node.B = &val
		}
		return node
	},
	"llm": func(p map[string]any) NodeExecutor {
		node := LLMNode{}
		if val, ok := p["prompt"].(string); ok {
			node.Prompt = &val
		}
		return node
	},
	"safe_object": func(p map[string]any) NodeExecutor {
		node := SafeObjectNode{Schema: make(map[string]string)}
		if schema, ok := p["schema"].(map[string]any); ok {
			for k, v := range schema {
				if vStr, ok := v.(string); ok {
					node.Schema[k] = vStr
				}
			}
		}
		return node
	},
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

		params := config.Params
		if params == nil {
			params = map[string]any{}
		}
		e.Register(factory(params))
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
