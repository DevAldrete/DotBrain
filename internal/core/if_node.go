package core

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// IfNode evaluates a condition against the input data and routes execution
// accordingly. It works with the DAG engine's conditional edge system:
//   - On condition=true: output includes {"condition": true} and the node succeeds,
//     so "success" edges are followed.
//   - On condition=false: output includes {"condition": false} and the node fails
//     with a controlled error, so "failure" edges are followed.
//
// This allows If nodes to be wired with success/failure edges to implement
// branching logic (if/else) in the workflow DAG.
//
// Supported operators: ==, !=, >, <, >=, <=, contains, not_contains, exists, not_exists, is_empty, not_empty
//
// Params:
//   - field: the input field to evaluate (e.g. "status", "count")
//   - operator: comparison operator
//   - value: the value to compare against (not needed for exists/not_exists/is_empty/not_empty)
//   - mode: "pass_through" (default, forwards input as output) or "boolean" (outputs only condition result)
type IfNode struct {
	Field    string
	Operator string
	Value    string
	Mode     string // "pass_through" or "boolean"
}

// NewIfNode creates an IfNode from params.
func NewIfNode(params map[string]any) *IfNode {
	node := &IfNode{
		Operator: "==",
		Mode:     "pass_through",
	}

	if field, ok := params["field"].(string); ok {
		node.Field = field
	}
	if op, ok := params["operator"].(string); ok {
		node.Operator = op
	}
	if val, ok := params["value"].(string); ok {
		node.Value = val
	} else if val, ok := params["value"]; ok {
		node.Value = fmt.Sprintf("%v", val)
	}
	if mode, ok := params["mode"].(string); ok {
		node.Mode = mode
	}

	return node
}

// Execute evaluates the condition. On true, returns the input data (or boolean output).
// On false, returns a controlled error so the DAG engine follows failure edges.
func (n *IfNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	if n.Field == "" {
		return nil, fmt.Errorf("if node: missing required param 'field'")
	}

	result := n.evaluate(input)

	output := make(map[string]any)

	if n.Mode == "boolean" {
		output["condition"] = result
	} else {
		// pass_through: copy all input fields
		for k, v := range input {
			output[k] = v
		}
		output["condition"] = result
	}

	if result {
		return output, nil
	}

	// On false, return the output AND a controlled error.
	// The DAG engine stores the error output via the failure hook and
	// follows "failure" edges. This is the mechanism for if/else branching.
	return nil, &ConditionFalseError{Output: output}
}

// ConditionFalseError is a controlled error type that carries the output data
// from a false condition evaluation. The DAG engine can use this to still
// pass data downstream on failure edges.
type ConditionFalseError struct {
	Output map[string]any
}

func (e *ConditionFalseError) Error() string {
	return "condition evaluated to false"
}

// evaluate performs the actual condition check.
func (n *IfNode) evaluate(input map[string]any) bool {
	fieldVal, exists := input[n.Field]

	switch n.Operator {
	case "exists":
		return exists
	case "not_exists":
		return !exists
	case "is_empty":
		return !exists || isEmpty(fieldVal)
	case "not_empty":
		return exists && !isEmpty(fieldVal)
	}

	if !exists {
		return false
	}

	fieldStr := fmt.Sprintf("%v", fieldVal)

	switch n.Operator {
	case "==":
		return fieldStr == n.Value
	case "!=":
		return fieldStr != n.Value
	case ">":
		return compareNumeric(fieldVal, n.Value) > 0
	case "<":
		return compareNumeric(fieldVal, n.Value) < 0
	case ">=":
		return compareNumeric(fieldVal, n.Value) >= 0
	case "<=":
		return compareNumeric(fieldVal, n.Value) <= 0
	case "contains":
		return strings.Contains(fieldStr, n.Value)
	case "not_contains":
		return !strings.Contains(fieldStr, n.Value)
	default:
		return false
	}
}

// compareNumeric compares two values numerically. Returns -1, 0, or 1.
// If either value cannot be parsed as a number, falls back to string comparison.
func compareNumeric(a any, bStr string) int {
	aFloat, aErr := toFloat64(a)
	bFloat, bErr := strconv.ParseFloat(bStr, 64)

	if aErr != nil || bErr != nil {
		// Fall back to string comparison
		aStr := fmt.Sprintf("%v", a)
		return strings.Compare(aStr, bStr)
	}

	if aFloat < bFloat {
		return -1
	}
	if aFloat > bFloat {
		return 1
	}
	return 0
}

// toFloat64 attempts to convert a value to float64.
func toFloat64(v any) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

// isEmpty checks if a value is considered "empty".
func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == ""
	case []any:
		return len(val) == 0
	case map[string]any:
		return len(val) == 0
	default:
		return false
	}
}
