package core

import "encoding/json"

// WorkflowDefinition represents a sequence of nodes to execute
type WorkflowDefinition struct {
	Nodes []NodeConfig `json:"nodes"`
	Edges []EdgeConfig `json:"edges"`
}

// EdgeConfig represents a connection between two nodes
type EdgeConfig struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Condition string `json:"condition,omitempty"`
}

// NodeConfig specifies a single execution step
type NodeConfig struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Params      map[string]any `json:"params,omitempty"`
	RetryPolicy *RetryPolicy   `json:"retry_policy,omitempty"`
}

// RetryPolicy configures retry behavior for a node on failure.
type RetryPolicy struct {
	MaxAttempts     int     `json:"max_attempts"`        // total attempts including the first; default 1 (no retry)
	InitialInterval int     `json:"initial_interval_ms"` // milliseconds; default 1000
	BackoffFactor   float64 `json:"backoff_factor"`      // multiplier per attempt; default 2.0
	MaxInterval     int     `json:"max_interval_ms"`     // cap on backoff; default 30000 (30s)
}

// ParseDefinition parses the JSONB array from the database into a WorkflowDefinition
func ParseDefinition(data []byte) (*WorkflowDefinition, error) {
	var def WorkflowDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, err
	}
	return &def, nil
}
