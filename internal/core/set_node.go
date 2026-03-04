package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// SetNode sets, transforms, or removes variables in the data flow.
// It acts as a variable registry — allowing workflows to reshape data
// between nodes without writing custom code.
//
// Params:
//   - values: map of field names to values to set (supports {{input.field}} templates)
//   - mode: "merge" (default) merges new values into input, "replace" outputs only the set values,
//     "append" adds to existing arrays, "delete" removes specified fields
//   - rename: map of old field name -> new field name (renames fields in the output)
//   - cast: map of field name -> target type ("string", "number", "boolean", "json")
type SetNode struct {
	Values map[string]any
	Mode   string
	Rename map[string]string
	Cast   map[string]string
}

// NewSetNode creates a SetNode from params.
func NewSetNode(params map[string]any) *SetNode {
	node := &SetNode{
		Values: make(map[string]any),
		Mode:   "merge",
		Rename: make(map[string]string),
		Cast:   make(map[string]string),
	}

	if values, ok := params["values"].(map[string]any); ok {
		node.Values = values
	}
	if mode, ok := params["mode"].(string); ok {
		node.Mode = mode
	}
	if rename, ok := params["rename"].(map[string]any); ok {
		for k, v := range rename {
			if vStr, ok := v.(string); ok {
				node.Rename[k] = vStr
			}
		}
	}
	if cast, ok := params["cast"].(map[string]any); ok {
		for k, v := range cast {
			if vStr, ok := v.(string); ok {
				node.Cast[k] = vStr
			}
		}
	}

	return node
}

// Execute processes the input data according to the configured mode.
func (n *SetNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	var output map[string]any

	switch n.Mode {
	case "replace":
		output = n.executeReplace(input)
	case "append":
		output = n.executeAppend(input)
	case "delete":
		output = n.executeDelete(input)
	default: // "merge"
		output = n.executeMerge(input)
	}

	// Apply renames
	for oldKey, newKey := range n.Rename {
		if val, exists := output[oldKey]; exists {
			output[newKey] = val
			delete(output, oldKey)
		}
	}

	// Apply type casts
	for field, targetType := range n.Cast {
		if val, exists := output[field]; exists {
			casted, err := castValue(val, targetType)
			if err != nil {
				return nil, fmt.Errorf("set node: failed to cast field %q to %s: %w", field, targetType, err)
			}
			output[field] = casted
		}
	}

	return output, nil
}

// executeMerge merges set values into the input, overwriting existing fields.
func (n *SetNode) executeMerge(input map[string]any) map[string]any {
	output := make(map[string]any)
	for k, v := range input {
		output[k] = v
	}
	for k, v := range n.Values {
		output[k] = resolveSetValue(v, input)
	}
	return output
}

// executeReplace outputs only the set values, discarding input.
func (n *SetNode) executeReplace(input map[string]any) map[string]any {
	output := make(map[string]any)
	for k, v := range n.Values {
		output[k] = resolveSetValue(v, input)
	}
	return output
}

// executeAppend adds values to existing arrays or creates new arrays.
func (n *SetNode) executeAppend(input map[string]any) map[string]any {
	output := make(map[string]any)
	for k, v := range input {
		output[k] = v
	}
	for k, v := range n.Values {
		resolved := resolveSetValue(v, input)
		existing, exists := output[k]
		if exists {
			if arr, ok := existing.([]any); ok {
				output[k] = append(arr, resolved)
			} else {
				output[k] = []any{existing, resolved}
			}
		} else {
			output[k] = []any{resolved}
		}
	}
	return output
}

// executeDelete removes specified fields from the input.
func (n *SetNode) executeDelete(input map[string]any) map[string]any {
	output := make(map[string]any)
	for k, v := range input {
		output[k] = v
	}
	for k := range n.Values {
		delete(output, k)
	}
	return output
}

// resolveSetValue resolves template expressions in set values.
func resolveSetValue(v any, input map[string]any) any {
	if strVal, ok := v.(string); ok {
		resolved := ApplyTemplate(strVal, input)
		if resolved != strVal {
			return resolved
		}
		return strVal
	}
	return v
}

// castValue converts a value to the specified target type.
func castValue(v any, targetType string) (any, error) {
	switch targetType {
	case "string":
		return fmt.Sprintf("%v", v), nil
	case "number":
		switch val := v.(type) {
		case float64:
			return val, nil
		case string:
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot parse %q as number", val)
			}
			return f, nil
		case bool:
			if val {
				return float64(1), nil
			}
			return float64(0), nil
		default:
			return nil, fmt.Errorf("cannot cast %T to number", v)
		}
	case "boolean":
		switch val := v.(type) {
		case bool:
			return val, nil
		case string:
			return strings.ToLower(val) == "true" || val == "1", nil
		case float64:
			return val != 0, nil
		default:
			return v != nil, nil
		}
	case "json":
		switch val := v.(type) {
		case string:
			var parsed any
			if err := json.Unmarshal([]byte(val), &parsed); err != nil {
				return nil, fmt.Errorf("cannot parse %q as JSON", val)
			}
			return parsed, nil
		default:
			return val, nil
		}
	default:
		return nil, fmt.Errorf("unknown target type %q", targetType)
	}
}
