package core

import "encoding/json"

// WorkflowDefinition represents a sequence of nodes to execute
type WorkflowDefinition struct {
	Nodes []NodeConfig `json:"nodes"`
}

// NodeConfig specifies a single execution step
type NodeConfig struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Params map[string]any `json:"params,omitempty"`
}

// ParseDefinition parses the JSONB array from the database into a WorkflowDefinition
func ParseDefinition(data []byte) (*WorkflowDefinition, error) {
	var def WorkflowDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, err
	}
	return &def, nil
}
