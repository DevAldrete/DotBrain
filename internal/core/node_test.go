package core

import (
	"context"
	"reflect"
	"testing"
)

func TestEchoNode_Execute(t *testing.T) {
	node := EchoNode{}
	input := map[string]any{"key": "value", "num": 42}

	result, err := node.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !reflect.DeepEqual(result, input) {
		t.Errorf("expected result to equal input. got %v, want %v", result, input)
	}
}

func TestFailNode_Execute(t *testing.T) {
	node := FailNode{}

	result, err := node.Execute(context.Background(), nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "this node always fails" {
		t.Errorf("expected error message 'this node always fails', got '%v'", err.Error())
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestMathNode_Execute(t *testing.T) {
	node := MathNode{}

	tests := []struct {
		name       string
		input      map[string]any
		wantResult map[string]any
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid addition",
			input:      map[string]any{"a": 10.0, "b": 5.5},
			wantResult: map[string]any{"result": 15.5},
			wantErr:    false,
		},
		{
			name:       "missing 'a'",
			input:      map[string]any{"b": 5.5},
			wantResult: nil,
			wantErr:    true,
			errMsg:     "missing or invalid 'a' parameter",
		},
		{
			name:       "missing 'b'",
			input:      map[string]any{"a": 10.0},
			wantResult: nil,
			wantErr:    true,
			errMsg:     "missing or invalid 'b' parameter",
		},
		{
			name:       "invalid type for 'a'",
			input:      map[string]any{"a": "10", "b": 5.5},
			wantResult: nil,
			wantErr:    true,
			errMsg:     "missing or invalid 'a' parameter",
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
