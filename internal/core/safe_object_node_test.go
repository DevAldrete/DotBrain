package core

import (
	"context"
	"reflect"
	"testing"
)

func TestSafeObjectNode_Execute(t *testing.T) {
	// A simple schema: we require "name" to be a string, and "age" to be a float64
	schema := map[string]string{
		"name": "string",
		"age":  "float64",
	}

	node := SafeObjectNode{Schema: schema}

	tests := []struct {
		name       string
		input      map[string]any
		wantResult map[string]any
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid input",
			input:      map[string]any{"name": "Alice", "age": 30.0, "extra": "ignored"},
			wantResult: map[string]any{"name": "Alice", "age": 30.0}, // extra fields filtered
			wantErr:    false,
		},
		{
			name:       "missing required field",
			input:      map[string]any{"name": "Alice"},
			wantResult: nil,
			wantErr:    true,
			errMsg:     "missing required field: age",
		},
		{
			name:       "invalid type for field",
			input:      map[string]any{"name": "Alice", "age": "thirty"},
			wantResult: nil,
			wantErr:    true,
			errMsg:     "invalid type for field age: expected float64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := node.Execute(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if err.Error() != tt.errMsg {
					t.Errorf("expected error message '%s', got '%v'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if !reflect.DeepEqual(result, tt.wantResult) {
					t.Errorf("expected result %v, got %v", tt.wantResult, result)
				}
			}
		})
	}
}
