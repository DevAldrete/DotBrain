package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/devaldrete/dotbrain/internal/core"
)

func TestEngine_SequentialExecution(t *testing.T) {
	engine := core.NewEngine()

	engine.Register(core.EchoNode{})
	engine.Register(core.MathNode{})

	ctx := context.Background()
	input := map[string]any{
		"a": 5.0,
		"b": 10.0,
	}

	result, err := engine.Execute(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := result["result"].(float64)
	if !ok {
		t.Fatalf("expected result of type float64")
	}

	if val != 15.0 {
		t.Errorf("expected 15.0, got %v", val)
	}
}

func TestEngine_NodeFailure(t *testing.T) {
	engine := core.NewEngine()

	engine.Register(core.FailNode{})
	engine.Register(core.MathNode{})

	ctx := context.Background()
	input := map[string]any{
		"a": 5.0,
		"b": 10.0,
	}

	_, err := engine.Execute(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Error() != "node execution failed: this node always fails" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestEngine_LoadFromDefinition_PassesParamsToNode verifies that params
// defined in a NodeConfig are passed to the node at instantiation time.
func TestEngine_LoadFromDefinition_PassesParamsToNode(t *testing.T) {
	def := &core.WorkflowDefinition{
		Nodes: []core.NodeConfig{
			{
				ID:   "1",
				Type: "math",
				Params: map[string]any{
					"a": 10.0,
					"b": 20.0,
				},
			},
		},
	}

	engine := core.NewEngine()
	err := engine.LoadFromDefinition(def)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Execute with empty input to see if it uses the params
	result, err := engine.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := result["result"].(float64)
	if !ok || val != 30.0 {
		t.Errorf("expected 30.0, got %v", val)
	}
}

// TestEngine_LoadFromDefinition_NilParamsSafe verifies that a NodeConfig
// with no params field does not panic (passes an empty map instead of nil).
func TestEngine_LoadFromDefinition_NilParamsSafe(t *testing.T) {
	def := &core.WorkflowDefinition{
		Nodes: []core.NodeConfig{
			{ID: "1", Type: "echo"}, // no Params field
		},
	}
	engine := core.NewEngine()
	err := engine.LoadFromDefinition(def)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should execute echo node successfully
	result, err := engine.Execute(context.Background(), map[string]any{"hello": "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["hello"] != "world" {
		t.Errorf("expected world, got %v", result["hello"])
	}
}

type recordingHook struct {
	starts    []string
	completes []string
	failures  []string
	retries   []string
}

func (h *recordingHook) OnNodeStart(ctx context.Context, nodeID string, input map[string]any) {
	h.starts = append(h.starts, nodeID)
}

func (h *recordingHook) OnNodeComplete(ctx context.Context, nodeID string, output map[string]any) {
	h.completes = append(h.completes, nodeID)
}

func (h *recordingHook) OnNodeFail(ctx context.Context, nodeID string, err error) {
	h.failures = append(h.failures, nodeID)
}

func (h *recordingHook) OnNodeRetry(ctx context.Context, nodeID string, attempt int, err error) {
	h.retries = append(h.retries, nodeID)
}

// TestEngine_Execute_CallsHookForEachNode verifies that the lifecycle hook
// is called once per node in the correct order.
func TestEngine_Execute_CallsHookForEachNode(t *testing.T) {
	hook := &recordingHook{}
	engine := core.NewEngine()
	engine.Hook = hook
	engine.RegisterWithID("node1", core.EchoNode{})
	engine.RegisterWithID("node2", core.EchoNode{})

	input := map[string]any{"key": "value"}
	_, err := engine.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}

	if len(hook.starts) != 2 {
		t.Errorf("expected 2 OnNodeStart calls, got %d", len(hook.starts))
	}
	if len(hook.completes) != 2 {
		t.Errorf("expected 2 OnNodeComplete calls, got %d", len(hook.completes))
	}
	if hook.starts[0] != "node1" || hook.starts[1] != "node2" {
		t.Errorf("unexpected node start order: %v", hook.starts)
	}
}

// TestEngine_Execute_CallsOnNodeFail verifies that OnNodeFail is called
// when a node returns an error.
func TestEngine_Execute_CallsOnNodeFail(t *testing.T) {
	hook := &recordingHook{}
	engine := core.NewEngine()
	engine.Hook = hook
	engine.RegisterWithID("failNode", core.FailNode{})

	_, err := engine.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Fatal("expected error from FailNode")
	}
	if len(hook.failures) != 1 {
		t.Errorf("expected 1 OnNodeFail call, got %d", len(hook.failures))
	}
	if hook.failures[0] != "failNode" {
		t.Errorf("expected failure from failNode, got %v", hook.failures[0])
	}
}

// TestEngine_Execute_NilHookDoesNotPanic verifies that an engine with
// no hook set runs normally.
func TestEngine_Execute_NilHookDoesNotPanic(t *testing.T) {
	engine := core.NewEngine()
	engine.RegisterWithID("node1", core.EchoNode{})
	_, err := engine.Execute(context.Background(), map[string]any{"x": 1})
	if err != nil {
		t.Fatal(err)
	}
}

// TestEngine_DAG_LinearEdges verifies that explicit edges produce the same result as no-edges.
func TestEngine_DAG_LinearEdges(t *testing.T) {
	def := &core.WorkflowDefinition{
		Nodes: []core.NodeConfig{
			{ID: "A", Type: "math", Params: map[string]any{"a": 10.0, "b": 20.0}},
			{ID: "B", Type: "math", Params: map[string]any{"a": "{{input.result}}", "b": 5.0}},
		},
		Edges: []core.EdgeConfig{
			{From: "A", To: "B"},
		},
	}

	engine := core.NewEngine()
	engine.Register(core.MathNode{})

	err := engine.LoadFromDefinition(def)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	result, err := engine.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := result["result"].(float64)
	if !ok || val != 35.0 {
		t.Errorf("expected 35.0, got %v", val)
	}
}

// TestEngine_DAG_FanOut verifies that one source running to two targets executes both targets.
func TestEngine_DAG_FanOut(t *testing.T) {
	def := &core.WorkflowDefinition{
		Nodes: []core.NodeConfig{
			{ID: "start", Type: "echo"},
			{ID: "branchA", Type: "echo"},
			{ID: "branchB", Type: "echo"},
		},
		Edges: []core.EdgeConfig{
			{From: "start", To: "branchA"},
			{From: "start", To: "branchB"},
		},
	}

	engine := core.NewEngine()
	hook := &recordingHook{}
	engine.Hook = hook
	engine.Register(core.EchoNode{})

	err := engine.LoadFromDefinition(def)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = engine.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(hook.completes) != 3 {
		t.Fatalf("expected 3 nodes to complete, got %d", len(hook.completes))
	}
}

// TestEngine_DAG_ConditionalSuccess verifies that a success edge is followed, and failure edge is skipped.
func TestEngine_DAG_ConditionalSuccess(t *testing.T) {
	def := &core.WorkflowDefinition{
		Nodes: []core.NodeConfig{
			{ID: "start", Type: "echo"},
			{ID: "on-success", Type: "echo"},
			{ID: "on-failure", Type: "echo"},
		},
		Edges: []core.EdgeConfig{
			{From: "start", To: "on-success", Condition: "success"},
			{From: "start", To: "on-failure", Condition: "failure"},
		},
	}

	engine := core.NewEngine()
	hook := &recordingHook{}
	engine.Hook = hook
	engine.Register(core.EchoNode{})

	err := engine.LoadFromDefinition(def)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = engine.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundSuccess := false
	foundFailure := false
	for _, id := range hook.completes {
		if id == "on-success" {
			foundSuccess = true
		}
		if id == "on-failure" {
			foundFailure = true
		}
	}

	if !foundSuccess {
		t.Errorf("expected on-success node to execute")
	}
	if foundFailure {
		t.Errorf("expected on-failure node to be skipped")
	}
}

// TestEngine_DAG_ConditionalFailure verifies that a failure edge is followed after a node error.
func TestEngine_DAG_ConditionalFailure(t *testing.T) {
	def := &core.WorkflowDefinition{
		Nodes: []core.NodeConfig{
			{ID: "start", Type: "fail"},
			{ID: "on-success", Type: "echo"},
			{ID: "on-failure", Type: "echo"},
		},
		Edges: []core.EdgeConfig{
			{From: "start", To: "on-success", Condition: "success"},
			{From: "start", To: "on-failure", Condition: "failure"},
		},
	}

	engine := core.NewEngine()
	hook := &recordingHook{}
	engine.Hook = hook
	engine.Register(core.EchoNode{})
	engine.RegisterWithID("fail", core.FailNode{})

	err := engine.LoadFromDefinition(def)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = engine.Execute(context.Background(), map[string]any{})
	// Even though a node failed, the workflow engine should handle the failure edge without failing the whole execution immediately,
	// or the on-failure should be executed.

	foundSuccess := false
	foundFailure := false
	for _, id := range hook.completes {
		if id == "on-success" {
			foundSuccess = true
		}
		if id == "on-failure" {
			foundFailure = true
		}
	}

	if foundSuccess {
		t.Errorf("expected on-success node to be skipped")
	}
	if !foundFailure {
		t.Errorf("expected on-failure node to execute")
	}
}

// TestEngine_DAG_CycleDetection verifies that a cyclic definition returns an error at load time.
func TestEngine_DAG_CycleDetection(t *testing.T) {
	def := &core.WorkflowDefinition{
		Nodes: []core.NodeConfig{
			{ID: "A", Type: "echo"},
			{ID: "B", Type: "echo"},
		},
		Edges: []core.EdgeConfig{
			{From: "A", To: "B"},
			{From: "B", To: "A"},
		},
	}

	engine := core.NewEngine()
	engine.Register(core.EchoNode{})

	err := engine.LoadFromDefinition(def)
	if err == nil {
		t.Fatalf("expected error due to cycle, got nil")
	}
}

// TestEngine_DAG_BackwardCompat verifies that no edges field results in linear execution.
func TestEngine_DAG_BackwardCompat(t *testing.T) {
	def := &core.WorkflowDefinition{
		Nodes: []core.NodeConfig{
			{ID: "A", Type: "echo"},
			{ID: "B", Type: "echo"},
			{ID: "C", Type: "echo"},
		},
	}

	engine := core.NewEngine()
	hook := &recordingHook{}
	engine.Hook = hook
	engine.Register(core.EchoNode{})

	err := engine.LoadFromDefinition(def)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = engine.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(hook.completes) != 3 {
		t.Fatalf("expected 3 nodes to complete, got %d", len(hook.completes))
	}

	if hook.completes[0] != "A" || hook.completes[1] != "B" || hook.completes[2] != "C" {
		t.Errorf("unexpected execution order: %v", hook.completes)
	}
}

// --- Retry / Backoff Tests ---

// TestBackoffDuration verifies the exponential backoff calculation.
func TestBackoffDuration(t *testing.T) {
	policy := &core.RetryPolicy{
		MaxAttempts:     5,
		InitialInterval: 100,
		BackoffFactor:   2.0,
		MaxInterval:     1000,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 100 * time.Millisecond},  // 100 * 2^0 = 100
		{2, 200 * time.Millisecond},  // 100 * 2^1 = 200
		{3, 400 * time.Millisecond},  // 100 * 2^2 = 400
		{4, 800 * time.Millisecond},  // 100 * 2^3 = 800
		{5, 1000 * time.Millisecond}, // 100 * 2^4 = 1600, capped to 1000
	}

	for _, tc := range tests {
		got := core.BackoffDuration(policy, tc.attempt)
		if got != tc.expected {
			t.Errorf("attempt %d: expected %v, got %v", tc.attempt, tc.expected, got)
		}
	}
}

// TestEngine_Retry_SucceedsOnSecondAttempt verifies that a node with retry policy
// succeeds when it fails once then succeeds on the second attempt.
func TestEngine_Retry_SucceedsOnSecondAttempt(t *testing.T) {
	def := &core.WorkflowDefinition{
		Nodes: []core.NodeConfig{
			{
				ID:   "flaky",
				Type: "counting_fail",
				Params: map[string]any{
					"fail_times": 1.0, // fail once, then succeed
				},
				RetryPolicy: &core.RetryPolicy{
					MaxAttempts:     3,
					InitialInterval: 1, // 1ms for fast tests
					BackoffFactor:   1.0,
					MaxInterval:     10,
				},
			},
		},
	}

	engine := core.NewEngine()
	hook := &recordingHook{}
	engine.Hook = hook

	err := engine.LoadFromDefinition(def)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	result, err := engine.Execute(context.Background(), map[string]any{"value": "hello"})
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}

	if result["value"] != "hello" {
		t.Errorf("expected value=hello, got %v", result["value"])
	}

	// Should have 1 retry call (failed once before succeeding)
	if len(hook.retries) != 1 {
		t.Errorf("expected 1 OnNodeRetry call, got %d", len(hook.retries))
	}

	// Should have completed successfully
	if len(hook.completes) != 1 {
		t.Errorf("expected 1 OnNodeComplete call, got %d", len(hook.completes))
	}

	// Should NOT have a final failure
	if len(hook.failures) != 0 {
		t.Errorf("expected 0 OnNodeFail calls, got %d", len(hook.failures))
	}
}

// TestEngine_Retry_ExhaustsAllAttempts verifies that a node that always fails
// exhausts all retry attempts and then the run fails.
func TestEngine_Retry_ExhaustsAllAttempts(t *testing.T) {
	def := &core.WorkflowDefinition{
		Nodes: []core.NodeConfig{
			{
				ID:   "always-fail",
				Type: "fail",
				RetryPolicy: &core.RetryPolicy{
					MaxAttempts:     3,
					InitialInterval: 1,
					BackoffFactor:   1.0,
					MaxInterval:     10,
				},
			},
		},
	}

	engine := core.NewEngine()
	hook := &recordingHook{}
	engine.Hook = hook

	err := engine.LoadFromDefinition(def)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	_, err = engine.Execute(context.Background(), map[string]any{})

	// Should have 2 retry calls (attempt 1 and 2 trigger retries, attempt 3 is final failure)
	if len(hook.retries) != 2 {
		t.Errorf("expected 2 OnNodeRetry calls, got %d", len(hook.retries))
	}

	// Should have 1 final failure
	if len(hook.failures) != 1 {
		t.Errorf("expected 1 OnNodeFail call, got %d", len(hook.failures))
	}
}

// TestEngine_Retry_RespectsContext verifies that a cancelled context aborts the
// retry wait immediately rather than waiting for the full backoff duration.
func TestEngine_Retry_RespectsContext(t *testing.T) {
	def := &core.WorkflowDefinition{
		Nodes: []core.NodeConfig{
			{
				ID:   "slow-retry",
				Type: "fail",
				RetryPolicy: &core.RetryPolicy{
					MaxAttempts:     10,
					InitialInterval: 60000, // 60 seconds — we should NOT wait this long
					BackoffFactor:   1.0,
					MaxInterval:     60000,
				},
			},
		},
	}

	engine := core.NewEngine()

	err := engine.LoadFromDefinition(def)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = engine.Execute(ctx, map[string]any{})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}

	// Should have exited quickly (well under the 60s backoff)
	if elapsed > 2*time.Second {
		t.Errorf("retry did not respect context cancellation, took %v", elapsed)
	}
}

// TestEngine_Retry_NoPolicy_NoChange verifies that a node with no retry policy
// fails on the first error with no retry, identical to the existing behavior.
func TestEngine_Retry_NoPolicy_NoChange(t *testing.T) {
	def := &core.WorkflowDefinition{
		Nodes: []core.NodeConfig{
			{
				ID:   "no-retry",
				Type: "fail",
				// No RetryPolicy
			},
		},
	}

	engine := core.NewEngine()
	hook := &recordingHook{}
	engine.Hook = hook

	err := engine.LoadFromDefinition(def)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	_, err = engine.Execute(context.Background(), map[string]any{})

	// Should have 0 retries
	if len(hook.retries) != 0 {
		t.Errorf("expected 0 OnNodeRetry calls, got %d", len(hook.retries))
	}

	// Should have 1 failure
	if len(hook.failures) != 1 {
		t.Errorf("expected 1 OnNodeFail call, got %d", len(hook.failures))
	}
}
