package core

import (
	"context"
	"fmt"
)

// MergeNode combines data from multiple input branches into a single output.
// In a DAG workflow, when multiple branches converge, the engine merges
// predecessor outputs (last-write-wins). The MergeNode provides more control:
//
// Params:
//   - mode: "combine" (default) merges all fields with last-write-wins,
//     "append" collects values per key into arrays,
//     "zip" pairs values from branches by position,
//     "pick" selects specific fields from input
//   - fields: for "pick" mode, array of field names to keep
//   - prefix_branches: if true, prefix each field with its source branch ID (for combine mode)
type MergeNode struct {
	Mode           string
	Fields         []string
	PrefixBranches bool
}

// NewMergeNode creates a MergeNode from params.
func NewMergeNode(params map[string]any) *MergeNode {
	node := &MergeNode{
		Mode: "combine",
	}

	if mode, ok := params["mode"].(string); ok {
		node.Mode = mode
	}
	if fields, ok := params["fields"].([]any); ok {
		for _, f := range fields {
			if fStr, ok := f.(string); ok {
				node.Fields = append(node.Fields, fStr)
			}
		}
	}
	if prefix, ok := params["prefix_branches"].(bool); ok {
		node.PrefixBranches = prefix
	}

	return node
}

// Execute merges the input data according to the configured mode.
// The input comes from the engine's buildMergedInput which already combined
// all predecessor outputs. This node provides additional merge strategies.
func (n *MergeNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	switch n.Mode {
	case "append":
		return n.executeAppend(input)
	case "pick":
		return n.executePick(input)
	case "wait":
		return n.executeWait(input)
	default: // "combine"
		return n.executeCombine(input)
	}
}

// executeCombine passes through all merged input. Since the engine already
// merges predecessor outputs, this is essentially a passthrough that
// explicitly signals "I accept all inputs."
func (n *MergeNode) executeCombine(input map[string]any) (map[string]any, error) {
	output := make(map[string]any)
	for k, v := range input {
		output[k] = v
	}
	return output, nil
}

// executeAppend takes input fields that are arrays and concatenates them.
// Non-array values with the same key are collected into arrays.
func (n *MergeNode) executeAppend(input map[string]any) (map[string]any, error) {
	output := make(map[string]any)

	for k, v := range input {
		existing, exists := output[k]
		if !exists {
			output[k] = v
			continue
		}

		// Both existing and new value exist — merge them
		existArr, existIsArr := existing.([]any)
		newArr, newIsArr := v.([]any)

		switch {
		case existIsArr && newIsArr:
			output[k] = append(existArr, newArr...)
		case existIsArr:
			output[k] = append(existArr, v)
		case newIsArr:
			output[k] = append([]any{existing}, newArr...)
		default:
			output[k] = []any{existing, v}
		}
	}

	return output, nil
}

// executePick selects only the specified fields from the input.
func (n *MergeNode) executePick(input map[string]any) (map[string]any, error) {
	if len(n.Fields) == 0 {
		return nil, fmt.Errorf("merge node: 'pick' mode requires 'fields' parameter")
	}

	output := make(map[string]any)
	for _, field := range n.Fields {
		if val, exists := input[field]; exists {
			output[field] = val
		}
	}

	return output, nil
}

// executeWait is a synchronization point — it waits for all inputs to be
// available and passes them through. Functionally identical to combine,
// but semantically signals "wait for all branches before continuing."
func (n *MergeNode) executeWait(input map[string]any) (map[string]any, error) {
	output := make(map[string]any)
	for k, v := range input {
		output[k] = v
	}
	output["_merged"] = true
	return output, nil
}
