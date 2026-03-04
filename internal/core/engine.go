package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

var nodeRegistry = map[string]func(map[string]any) NodeExecutor{
	"echo": func(p map[string]any) NodeExecutor { return EchoNode{} },
	"fail": func(p map[string]any) NodeExecutor { return FailNode{} },
	"counting_fail": func(p map[string]any) NodeExecutor {
		failTimes := 0
		if v, ok := p["fail_times"].(float64); ok {
			failTimes = int(v)
		}
		return &CountingFailNode{FailTimes: failTimes}
	},
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
	OnNodeRetry(ctx context.Context, nodeID string, attempt int, err error)
}

type registeredNode struct {
	id       string
	executor NodeExecutor
}

// dagNode represents a node in the DAG with its executor, dependencies, and outgoing edges.
type dagNode struct {
	id          string
	executor    NodeExecutor
	params      map[string]any
	retryPolicy *RetryPolicy
}

// Engine is an orchestrator that executes nodes either sequentially or as a DAG.
type Engine struct {
	// Legacy sequential mode
	nodes []registeredNode

	// DAG mode
	dagNodes map[string]*dagNode
	edges    []EdgeConfig
	dagMode  bool

	Hook NodeLifecycleHook
}

// NewEngine creates a new Engine instance.
func NewEngine() *Engine {
	return &Engine{
		nodes: make([]registeredNode, 0),
	}
}

// inferEdges generates linear edges from node order when no edges are defined.
func inferEdges(nodes []NodeConfig) []EdgeConfig {
	if len(nodes) <= 1 {
		return nil
	}
	edges := make([]EdgeConfig, 0, len(nodes)-1)
	for i := 1; i < len(nodes); i++ {
		edges = append(edges, EdgeConfig{
			From: nodes[i-1].ID,
			To:   nodes[i].ID,
		})
	}
	return edges
}

// detectCycle checks for cycles in the DAG using Kahn's algorithm.
// Returns an error if a cycle is detected.
func detectCycle(nodeIDs []string, edges []EdgeConfig) error {
	inDegree := make(map[string]int)
	for _, id := range nodeIDs {
		inDegree[id] = 0
	}

	adjacency := make(map[string][]string)
	for _, edge := range edges {
		adjacency[edge.From] = append(adjacency[edge.From], edge.To)
		inDegree[edge.To]++
	}

	queue := make([]string, 0)
	for _, id := range nodeIDs {
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}

	visited := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		visited++

		for _, neighbor := range adjacency[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if visited != len(nodeIDs) {
		return fmt.Errorf("workflow definition contains a cycle")
	}
	return nil
}

// LoadFromDefinition instantiates nodes from a workflow definition and registers them.
// When edges are present (or inferred), the engine switches to DAG mode.
func (e *Engine) LoadFromDefinition(def *WorkflowDefinition) error {
	edges := def.Edges
	if len(edges) == 0 {
		edges = inferEdges(def.Nodes)
	}

	// Collect node IDs for cycle detection
	nodeIDs := make([]string, len(def.Nodes))
	for i, config := range def.Nodes {
		nodeIDs[i] = config.ID
	}

	// Detect cycles
	if len(edges) > 0 {
		if err := detectCycle(nodeIDs, edges); err != nil {
			return err
		}
	}

	// Build DAG nodes
	dagNodes := make(map[string]*dagNode, len(def.Nodes))
	for _, config := range def.Nodes {
		factory, exists := nodeRegistry[config.Type]
		if !exists {
			return fmt.Errorf("unknown node type: %s", config.Type)
		}

		params := config.Params
		if params == nil {
			params = map[string]any{}
		}
		dagNodes[config.ID] = &dagNode{
			id:          config.ID,
			executor:    factory(params),
			params:      params,
			retryPolicy: config.RetryPolicy,
		}
	}

	e.dagNodes = dagNodes
	e.edges = edges
	e.dagMode = true

	return nil
}

// Register adds a NodeExecutor to the engine's execution sequence (legacy mode).
func (e *Engine) Register(node NodeExecutor) {
	e.RegisterWithID("", node)
}

// RegisterWithID adds a NodeExecutor to the engine's execution sequence with a specific ID (legacy mode).
func (e *Engine) RegisterWithID(id string, node NodeExecutor) {
	e.nodes = append(e.nodes, registeredNode{
		id:       id,
		executor: node,
	})
}

// Execute runs the engine. If LoadFromDefinition was called (DAG mode), it runs
// the DAG executor. Otherwise, it falls back to sequential execution.
func (e *Engine) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	if e.dagMode {
		return e.executeDAG(ctx, input)
	}
	return e.executeSequential(ctx, input)
}

// executeSequential runs nodes in the legacy sequential order.
func (e *Engine) executeSequential(ctx context.Context, input map[string]any) (map[string]any, error) {
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

		currentData = output
	}

	return currentData, nil
}

// nodeResult holds the result of executing a single DAG node.
type nodeResult struct {
	id     string
	output map[string]any
	err    error
}

// executeDAG runs nodes in topological order with support for fan-out,
// fan-in, and conditional routing.
func (e *Engine) executeDAG(ctx context.Context, input map[string]any) (map[string]any, error) {
	// Build adjacency and in-degree maps
	outEdges := make(map[string][]EdgeConfig)
	inDegree := make(map[string]int)

	for id := range e.dagNodes {
		inDegree[id] = 0
	}
	for _, edge := range e.edges {
		outEdges[edge.From] = append(outEdges[edge.From], edge)
		inDegree[edge.To]++
	}

	// Track node outputs for data passing
	nodeOutputs := make(map[string]map[string]any)

	// Track remaining in-degrees (mutable during execution)
	remaining := make(map[string]int)
	for id, deg := range inDegree {
		remaining[id] = deg
	}

	// Find initial ready nodes (zero in-degree)
	var ready []string
	for id, deg := range remaining {
		if deg == 0 {
			ready = append(ready, id)
		}
	}

	// Track which nodes were deactivated (condition not met)
	deactivated := make(map[string]bool)

	var lastOutput map[string]any
	var terminalErr error // tracks unhandled node failures

	// Process nodes in topological batches
	for len(ready) > 0 {
		batch := ready
		ready = nil

		resultsCh := make(chan nodeResult, len(batch))
		var wg sync.WaitGroup

		for _, nodeID := range batch {
			if deactivated[nodeID] {
				// This node was deactivated by a condition check. Still need to
				// propagate deactivation to downstream nodes.
				e.propagateDeactivation(nodeID, outEdges, remaining, deactivated, &ready)
				continue
			}

			node := e.dagNodes[nodeID]

			// Build merged input from all predecessors
			mergedInput := e.buildMergedInput(nodeID, input, nodeOutputs)

			// Resolve template parameters against merged input
			resolvedInput := e.resolveTemplates(node.params, mergedInput)

			wg.Add(1)
			go func(n *dagNode, nodeInput map[string]any) {
				defer wg.Done()

				if e.Hook != nil {
					e.Hook.OnNodeStart(ctx, n.id, nodeInput)
				}

				output, err := e.executeWithRetry(ctx, n, nodeInput)
				resultsCh <- nodeResult{id: n.id, output: output, err: err}
			}(node, resolvedInput)
		}

		// Wait for all goroutines then close channel
		go func() {
			wg.Wait()
			close(resultsCh)
		}()

		for result := range resultsCh {
			nodeID := result.id
			succeeded := result.err == nil

			if succeeded {
				nodeOutputs[nodeID] = result.output
				lastOutput = result.output
				if e.Hook != nil {
					e.Hook.OnNodeComplete(ctx, nodeID, result.output)
				}
			} else {
				if e.Hook != nil {
					e.Hook.OnNodeFail(ctx, nodeID, result.err)
				}

				// Check if this failure is handled by a failure edge
				hasFailureEdge := false
				for _, edge := range outEdges[nodeID] {
					if edge.Condition == "failure" {
						hasFailureEdge = true
						break
					}
				}
				if !hasFailureEdge {
					terminalErr = fmt.Errorf("node %s failed: %w", nodeID, result.err)
				}
			}

			// Process outgoing edges based on conditions
			for _, edge := range outEdges[nodeID] {
				shouldFollow := false
				switch edge.Condition {
				case "":
					// Unconditional: always follow
					shouldFollow = succeeded
				case "success":
					shouldFollow = succeeded
				case "failure":
					shouldFollow = !succeeded
				}

				if !shouldFollow {
					deactivated[edge.To] = true
				}

				remaining[edge.To]--
				if remaining[edge.To] == 0 {
					ready = append(ready, edge.To)
				}
			}
		}
	}

	if terminalErr != nil {
		return nil, terminalErr
	}
	if lastOutput != nil {
		return lastOutput, nil
	}
	return input, nil
}

// propagateDeactivation handles nodes that were deactivated by conditions,
// decrementing in-degrees of their successors and potentially deactivating them too.
func (e *Engine) propagateDeactivation(nodeID string, outEdges map[string][]EdgeConfig, remaining map[string]int, deactivated map[string]bool, ready *[]string) {
	for _, edge := range outEdges[nodeID] {
		deactivated[edge.To] = true
		remaining[edge.To]--
		if remaining[edge.To] == 0 {
			*ready = append(*ready, edge.To)
		}
	}
}

// buildMergedInput creates the input map for a node by merging all predecessor outputs.
// If the node has no predecessors, it receives the original workflow input.
func (e *Engine) buildMergedInput(nodeID string, originalInput map[string]any, nodeOutputs map[string]map[string]any) map[string]any {
	// Find all predecessors of this node
	predecessors := make([]string, 0)
	for _, edge := range e.edges {
		if edge.To == nodeID {
			predecessors = append(predecessors, edge.From)
		}
	}

	if len(predecessors) == 0 {
		return originalInput
	}

	// Merge all predecessor outputs
	merged := make(map[string]any)
	for _, predID := range predecessors {
		if output, ok := nodeOutputs[predID]; ok {
			for k, v := range output {
				merged[k] = v
			}
		}
	}

	// If no predecessor produced output (e.g., all failed), use original input
	if len(merged) == 0 {
		return originalInput
	}

	return merged
}

// resolveTemplates resolves {{input.field}} templates in node params against the actual input.
func (e *Engine) resolveTemplates(params map[string]any, input map[string]any) map[string]any {
	resolved := make(map[string]any)
	for k, v := range input {
		resolved[k] = v
	}

	// Apply template resolution from params to the input
	for k, v := range params {
		if strVal, ok := v.(string); ok {
			resolvedVal := ApplyTemplate(strVal, input)
			// Try to parse as float64 if it looks like a number
			if resolvedVal != strVal {
				var f float64
				if _, err := fmt.Sscanf(resolvedVal, "%f", &f); err == nil {
					resolved[k] = f
				} else {
					resolved[k] = resolvedVal
				}
			} else {
				resolved[k] = v
			}
		} else {
			resolved[k] = v
		}
	}

	return resolved
}

// executeWithRetry runs a node's executor, retrying according to its RetryPolicy.
// If the node has no retry policy, it executes once (original behavior).
// On each failed attempt (except the last), it calls OnNodeRetry and waits for
// the backoff duration, respecting context cancellation.
func (e *Engine) executeWithRetry(ctx context.Context, n *dagNode, input map[string]any) (map[string]any, error) {
	maxAttempts := 1
	if n.retryPolicy != nil && n.retryPolicy.MaxAttempts > 1 {
		maxAttempts = n.retryPolicy.MaxAttempts
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		output, err := n.executor.Execute(ctx, input)
		if err == nil {
			return output, nil
		}

		lastErr = err

		// If this was the last attempt, don't retry
		if attempt == maxAttempts {
			break
		}

		// Fire retry hook
		if e.Hook != nil {
			e.Hook.OnNodeRetry(ctx, n.id, attempt, err)
		}

		// Wait for backoff, respecting context cancellation
		backoff := BackoffDuration(n.retryPolicy, attempt)
		select {
		case <-time.After(backoff):
			// continue to next attempt
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, lastErr
}

// BackoffDuration calculates the backoff wait time for a given retry attempt.
// attempt is 1-indexed (attempt 1 = first retry after initial failure).
// Formula: min(initialInterval * backoffFactor^(attempt-1), maxInterval).
func BackoffDuration(policy *RetryPolicy, attempt int) time.Duration {
	base := float64(policy.InitialInterval)
	factor := policy.BackoffFactor

	// Calculate: initialInterval * factor^(attempt-1)
	multiplier := 1.0
	for i := 1; i < attempt; i++ {
		multiplier *= factor
	}
	ms := base * multiplier

	// Cap at MaxInterval
	if max := float64(policy.MaxInterval); ms > max {
		ms = max
	}

	return time.Duration(ms) * time.Millisecond
}
