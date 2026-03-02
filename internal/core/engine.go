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
		return NewLLMNode(p)
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
	"http": func(p map[string]any) NodeExecutor {
		return NewHttpNode(p)
	},
}

// NodeLifecycleHook provides callbacks during node execution.
type NodeLifecycleHook interface {
	OnNodeStart(ctx context.Context, nodeID string, input map[string]any)
	OnNodeComplete(ctx context.Context, nodeID string, output map[string]any)
	OnNodeFail(ctx context.Context, nodeID string, err error)
}

type registeredNode struct {
	id       string
	executor NodeExecutor
}

// Engine is a simple orchestrator that executes a slice of NodeExecutors sequentially.
type Engine struct {
	nodes []registeredNode
	Hook  NodeLifecycleHook
}

// NewEngine creates a new Engine instance.
func NewEngine() *Engine {
	return &Engine{
		nodes: make([]registeredNode, 0),
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
		e.RegisterWithID(config.ID, factory(params))
	}
	return nil
}

// Register adds a NodeExecutor to the engine's execution sequence.
func (e *Engine) Register(node NodeExecutor) {
	e.RegisterWithID("", node)
}

// RegisterWithID adds a NodeExecutor to the engine's execution sequence with a specific ID.
func (e *Engine) RegisterWithID(id string, node NodeExecutor) {
	e.nodes = append(e.nodes, registeredNode{
		id:       id,
		executor: node,
	})
}

// Execute runs the registered nodes sequentially, passing the output of one
// as the input to the next.
func (e *Engine) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	currentData := input

	for _, nodeInfo := range e.nodes {
		if e.Hook != nil {
			e.Hook.OnNodeStart(ctx, nodeInfo.id, currentData)
		}

		output, err := nodeInfo.executor.Execute(ctx, currentData)
		if err != nil {
			if e.Hook != nil {
				e.Hook.OnNodeFail(ctx, nodeInfo.id, err)
			}
			return nil, fmt.Errorf("node execution failed: %w", err)
		}

		if e.Hook != nil {
			e.Hook.OnNodeComplete(ctx, nodeInfo.id, output)
		}

		// Optional: We can merge outputs or replace them. The simple requirement
		// says "passing output of Node A as input to Node B". Let's replace for now,
		// but typically we'd merge if the orchestrator is passing the accumulated state.
		// For a simple sequence where node B only relies on node A's output, replacement works.
		currentData = output
	}

	return currentData, nil
}
