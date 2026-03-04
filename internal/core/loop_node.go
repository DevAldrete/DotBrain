package core

import (
	"context"
	"fmt"
)

// LoopNode iterates over an array field from the input and collects the items
// along with iteration metadata. This node does NOT execute sub-workflows
// per iteration (that would require engine-level loop support). Instead, it
// "unrolls" an array so downstream nodes can process each item.
//
// For simple use cases, it outputs the full array with indices.
// For more complex iteration, use multiple Loop nodes chained with other nodes.
//
// Params:
//   - source_field: the input field containing the array to iterate (default: "items")
//   - item_field: the output field name for the current iteration context (default: "item")
//   - mode: "collect" (default) returns all items with metadata, "flatten" merges all items into output
type LoopNode struct {
	SourceField string
	ItemField   string
	Mode        string
}

// NewLoopNode creates a LoopNode from params.
func NewLoopNode(params map[string]any) *LoopNode {
	node := &LoopNode{
		SourceField: "items",
		ItemField:   "item",
		Mode:        "collect",
	}

	if sf, ok := params["source_field"].(string); ok {
		node.SourceField = sf
	}
	if itf, ok := params["item_field"].(string); ok {
		node.ItemField = itf
	}
	if mode, ok := params["mode"].(string); ok {
		node.Mode = mode
	}

	return node
}

// Execute processes the array. In "collect" mode, it outputs all items with
// metadata (index, total, first/last flags). In "flatten" mode, it merges
// all map items into a single output.
func (n *LoopNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	rawItems, exists := input[n.SourceField]
	if !exists {
		return nil, fmt.Errorf("loop node: source field %q not found in input", n.SourceField)
	}

	items, ok := rawItems.([]any)
	if !ok {
		return nil, fmt.Errorf("loop node: source field %q is not an array (got %T)", n.SourceField, rawItems)
	}

	if len(items) == 0 {
		return map[string]any{
			"items": []any{},
			"count": float64(0),
			"empty": true,
		}, nil
	}

	switch n.Mode {
	case "flatten":
		return n.executeFlatten(ctx, input, items)
	default:
		return n.executeCollect(ctx, input, items)
	}
}

// executeCollect returns all items with iteration metadata.
func (n *LoopNode) executeCollect(ctx context.Context, input map[string]any, items []any) (map[string]any, error) {
	total := len(items)
	collected := make([]any, 0, total)

	for i, item := range items {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		entry := map[string]any{
			n.ItemField: item,
			"index":     float64(i),
			"total":     float64(total),
			"is_first":  i == 0,
			"is_last":   i == total-1,
		}
		collected = append(collected, entry)
	}

	return map[string]any{
		"items": collected,
		"count": float64(total),
		"empty": false,
	}, nil
}

// executeFlatten merges all map items' fields into a single output, with
// later items overwriting earlier ones. Non-map items are collected in an array.
func (n *LoopNode) executeFlatten(ctx context.Context, input map[string]any, items []any) (map[string]any, error) {
	result := make(map[string]any)
	var nonMapItems []any

	for _, item := range items {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if m, ok := item.(map[string]any); ok {
			for k, v := range m {
				result[k] = v
			}
		} else {
			nonMapItems = append(nonMapItems, item)
		}
	}

	if len(nonMapItems) > 0 {
		result["values"] = nonMapItems
	}

	result["count"] = float64(len(items))
	result["empty"] = false

	return result, nil
}
