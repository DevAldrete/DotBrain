package core

import (
	"context"
	"fmt"
)

// SafeObjectNode validates an input map against a required schema,
// ensuring both presence and type match. It filters out extra fields.
type SafeObjectNode struct {
	Schema map[string]string // Maps field name to expected type (e.g., "string", "float64")
}

// Execute validates the input against the schema
func (n SafeObjectNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	result := make(map[string]any)

	for key, expectedType := range n.Schema {
		val, exists := input[key]
		if !exists {
			return nil, fmt.Errorf("missing required field: %s", key)
		}

		// Simple type checking
		switch expectedType {
		case "string":
			if _, ok := val.(string); !ok {
				return nil, fmt.Errorf("invalid type for field %s: expected string", key)
			}
		case "float64":
			if _, ok := val.(float64); !ok {
				return nil, fmt.Errorf("invalid type for field %s: expected float64", key)
			}
		default:
			return nil, fmt.Errorf("unsupported schema type: %s", expectedType)
		}

		result[key] = val
	}

	return result, nil
}
