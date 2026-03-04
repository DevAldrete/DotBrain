package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	db "github.com/devaldrete/dotbrain/internal/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestHealthCheckHandler(t *testing.T) {
	router := NewAPI(nil).NewRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/health", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["status"] != "UP" {
		t.Errorf("expected status 'UP', got '%s'", response["status"])
	}

	if response["timestamp"] == "" {
		t.Errorf("expected a timestamp, got empty string")
	}
}

func TestReadinessHandler(t *testing.T) {
	t.Skip("skipping db ping test")
	router := NewAPI(nil).NewRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/readiness", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["status"] != "READY" {
		t.Errorf("expected status 'READY', got '%s'", response["status"])
	}
}

func TestPingHandler(t *testing.T) {
	router := NewAPI(nil).NewRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/ping", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["message"] != "pong" {
		t.Errorf("expected message 'pong', got '%s'", response["message"])
	}
}

func TestWorkflowTriggerHandler_UnknownID(t *testing.T) {
	t.Skip("skipping db dependent test")
}

func TestWorkflowTriggerHandler_Success(t *testing.T) {
	t.Skip("skipping db dependent test")
}

// --- Workflow CRUD Tests ---

// TestUpdateWorkflow_Success verifies that PUT /api/v1/workflows/:id updates
// a workflow's name, description, and definition, returning the updated object.
func TestUpdateWorkflow_Success(t *testing.T) {
	recorder := &queryRecorder{
		workflowDef: []byte(`{"nodes":[{"id":"1","type":"echo"}]}`),
	}
	queries := db.New(recorder)
	api := &API{queries: queries}
	router := api.NewRouter()

	body := strings.NewReader(`{
		"name": "Updated Pipeline",
		"description": "Updated desc",
		"definition": {"nodes":[{"id":"2","type":"log"}]}
	}`)
	req, _ := http.NewRequest("PUT", "/api/v1/workflows/01961234-5678-7000-8000-000000000001", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify UpdateWorkflow SQL was called with the correct args
	call := recorder.findCall("UPDATE workflows")
	if call == nil {
		t.Fatal("expected UpdateWorkflow query to be called")
	}

	// args: id, name, description, definition
	name, ok := call.args[1].(string)
	if !ok {
		t.Fatalf("expected name arg to be string, got %T", call.args[1])
	}
	if name != "Updated Pipeline" {
		t.Errorf("expected name 'Updated Pipeline', got %q", name)
	}

	desc, ok := call.args[2].(string)
	if !ok {
		t.Fatalf("expected description arg to be string, got %T", call.args[2])
	}
	if desc != "Updated desc" {
		t.Errorf("expected description 'Updated desc', got %q", desc)
	}
}

// TestUpdateWorkflow_NotFound verifies that PUT on a non-existent ID returns 404.
func TestUpdateWorkflow_NotFound(t *testing.T) {
	recorder := &queryRecorder{
		errQueries: []string{"UPDATE workflows"},
	}
	queries := db.New(recorder)
	api := &API{queries: queries}
	router := api.NewRouter()

	body := strings.NewReader(`{
		"name": "Updated",
		"description": "Desc",
		"definition": {"nodes":[]}
	}`)
	req, _ := http.NewRequest("PUT", "/api/v1/workflows/01961234-5678-7000-8000-000000000099", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestDeleteWorkflow_Success verifies that DELETE /api/v1/workflows/:id removes
// the workflow and returns 204.
func TestDeleteWorkflow_Success(t *testing.T) {
	recorder := &queryRecorder{
		workflowDef: []byte(`{"nodes":[]}`),
	}
	queries := db.New(recorder)
	api := &API{queries: queries}
	router := api.NewRouter()

	req, _ := http.NewRequest("DELETE", "/api/v1/workflows/01961234-5678-7000-8000-000000000001", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify GetWorkflow was called (existence check)
	getCall := recorder.findCall("FROM workflows")
	if getCall == nil {
		t.Fatal("expected GetWorkflow to be called for existence check")
	}

	// Verify DeleteWorkflow was called
	deleteCall := recorder.findCall("DELETE FROM workflows")
	if deleteCall == nil {
		t.Fatal("expected DeleteWorkflow query to be called")
	}
}

// TestDeleteWorkflow_NotFound verifies that DELETE on a non-existent ID returns 404.
func TestDeleteWorkflow_NotFound(t *testing.T) {
	recorder := &queryRecorder{
		errQueries: []string{"FROM workflows"},
	}
	queries := db.New(recorder)
	api := &API{queries: queries}
	router := api.NewRouter()

	req, _ := http.NewRequest("DELETE", "/api/v1/workflows/01961234-5678-7000-8000-000000000099", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	// Verify no DELETE was attempted
	deleteCall := recorder.findCall("DELETE FROM workflows")
	if deleteCall != nil {
		t.Error("DeleteWorkflow should not be called when workflow doesn't exist")
	}
}

// --- Run Endpoints Tests ---

// TestListWorkflowRuns_InvalidID verifies a bad workflow ID returns 400.
func TestListWorkflowRuns_InvalidID(t *testing.T) {
	router := NewAPI(nil).NewRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/workflows/not-a-uuid/runs", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestListWorkflowRuns_ReturnsEmptyArray verifies empty results return []
// instead of null.
func TestListWorkflowRuns_ReturnsEmptyArray(t *testing.T) {
	recorder := &queryRecorder{}
	queries := db.New(recorder)
	api := &API{queries: queries}
	router := api.NewRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/workflows/01961234-5678-7000-8000-000000000001/runs", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Must return [] not null
	body := strings.TrimSpace(w.Body.String())
	if body != "[]" {
		t.Errorf("expected empty array '[]', got %q", body)
	}
}

// TestGetRun_InvalidID verifies a bad run ID returns 400.
func TestGetRun_InvalidID(t *testing.T) {
	router := NewAPI(nil).NewRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/runs/not-a-uuid", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestListNodeExecutionsForRun_InvalidID verifies a bad run ID returns 400.
func TestListNodeExecutionsForRun_InvalidID(t *testing.T) {
	router := NewAPI(nil).NewRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/runs/not-a-uuid/nodes", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestListNodeExecutionsForRun_ReturnsEmptyArray verifies empty results return [].
func TestListNodeExecutionsForRun_ReturnsEmptyArray(t *testing.T) {
	recorder := &queryRecorder{}
	queries := db.New(recorder)
	api := &API{queries: queries}
	router := api.NewRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/runs/01961234-5678-7000-8000-000000000001/nodes", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := strings.TrimSpace(w.Body.String())
	if body != "[]" {
		t.Errorf("expected empty array '[]', got %q", body)
	}
}

// TestTriggerHandler_CreatesRunAsPending verifies the trigger handler creates
// a workflow run with status "pending" and then transitions to "running" with
// started_at set before execution begins.
func TestTriggerHandler_CreatesRunAsPending(t *testing.T) {
	recorder := &queryRecorder{}
	queries := db.New(recorder)
	api := &API{queries: queries}

	// We need GetWorkflow to succeed, so configure the recorder to return
	// a scannable workflow row on the first QueryRow call
	recorder.workflowDef = []byte(`{"nodes":[{"id":"1","type":"echo"}]}`)

	router := api.NewRouter()

	body := strings.NewReader(`{"input": "test"}`)
	req, _ := http.NewRequest("POST", "/api/v1/workflows/01961234-5678-7000-8000-000000000001/trigger", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	// Give the goroutine time to execute
	time.Sleep(100 * time.Millisecond)

	// Find the CreateWorkflowRun call and verify status is "pending"
	createRunCall := recorder.findCall("INSERT INTO workflow_runs")
	if createRunCall == nil {
		t.Fatal("expected CreateWorkflowRun to be called")
	}

	// Status is the 3rd arg (index 2): id, workflow_id, status, input_data
	status, ok := createRunCall.args[2].(string)
	if !ok {
		t.Fatalf("expected status arg to be string, got %T", createRunCall.args[2])
	}
	if status != "pending" {
		t.Errorf("expected run to be created with status 'pending', got %q", status)
	}

	// Find the first UpdateWorkflowRunStatus call — should transition to "running" with started_at
	updateCalls := recorder.findAllCalls("UPDATE workflow_runs")
	if len(updateCalls) == 0 {
		t.Fatal("expected at least one UpdateWorkflowRunStatus call")
	}

	// First update should be the "running" transition
	firstUpdate := updateCalls[0]
	runningStatus, ok := firstUpdate.args[1].(string)
	if !ok {
		t.Fatalf("expected status arg to be string, got %T", firstUpdate.args[1])
	}
	if runningStatus != "running" {
		t.Errorf("expected first update to set status 'running', got %q", runningStatus)
	}

	// started_at should be set (index 4): id, status, output_data, error, started_at, completed_at
	startedAt, ok := firstUpdate.args[4].(pgtype.Timestamptz)
	if !ok {
		t.Fatalf("expected started_at arg to be pgtype.Timestamptz, got %T", firstUpdate.args[4])
	}
	if !startedAt.Valid {
		t.Error("expected started_at to be set (Valid=true) when transitioning to 'running'")
	}
}

// queryRecorder captures all database calls for assertion in tests.
type queryRecorder struct {
	mu          sync.Mutex
	calls       []queryCall
	workflowDef []byte   // returned by GetWorkflow
	errQueries  []string // substrings; QueryRow matching these returns errRow (pgx.ErrNoRows)
}

type queryCall struct {
	query string
	args  []any
}

func (r *queryRecorder) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, queryCall{query: query, args: args})
	return pgconn.CommandTag{}, nil
}

func (r *queryRecorder) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, queryCall{query: query, args: args})
	return &emptyRows{}, nil
}

func (r *queryRecorder) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, queryCall{query: query, args: args})

	// If this query matches an error pattern, return a row that errors on Scan
	for _, pattern := range r.errQueries {
		if strings.Contains(query, pattern) {
			return &errRow{err: pgx.ErrNoRows}
		}
	}

	// If this is a GetWorkflow query, return a row that scans into a Workflow
	if strings.Contains(query, "FROM workflows") && r.workflowDef != nil {
		return &workflowRow{def: r.workflowDef}
	}

	return &mockRow{}
}

func (r *queryRecorder) findCall(querySubstring string) *queryCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, c := range r.calls {
		if strings.Contains(c.query, querySubstring) {
			return &r.calls[i]
		}
	}
	return nil
}

func (r *queryRecorder) findAllCalls(querySubstring string) []queryCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []queryCall
	for _, c := range r.calls {
		if strings.Contains(c.query, querySubstring) {
			result = append(result, c)
		}
	}
	return result
}

// workflowRow implements pgx.Row to return a fake Workflow for GetWorkflow.
type workflowRow struct {
	def []byte
}

func (r *workflowRow) Scan(dest ...any) error {
	// GetWorkflow scans: id, name, description, definition, created_at, updated_at
	if len(dest) >= 6 {
		if id, ok := dest[0].(*pgtype.UUID); ok {
			id.Bytes = [16]byte{1}
			id.Valid = true
		}
		if name, ok := dest[1].(*string); ok {
			*name = "test-workflow"
		}
		if desc, ok := dest[2].(*string); ok {
			*desc = "test"
		}
		if def, ok := dest[3].(*[]byte); ok {
			*def = r.def
		}
		if ts, ok := dest[4].(*pgtype.Timestamptz); ok {
			ts.Time = time.Now()
			ts.Valid = true
		}
		if ts, ok := dest[5].(*pgtype.Timestamptz); ok {
			ts.Time = time.Now()
			ts.Valid = true
		}
	}
	return nil
}

// emptyRows implements pgx.Rows for queries that return no results.
type emptyRows struct{}

func (r *emptyRows) Close()                                       {}
func (r *emptyRows) Err() error                                   { return nil }
func (r *emptyRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *emptyRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *emptyRows) Next() bool                                   { return false }
func (r *emptyRows) Scan(dest ...any) error                       { return nil }
func (r *emptyRows) Values() ([]any, error)                       { return nil, nil }
func (r *emptyRows) RawValues() [][]byte                          { return nil }
func (r *emptyRows) Conn() *pgx.Conn                              { return nil }

// errRow implements pgx.Row, returning an error from Scan (e.g., pgx.ErrNoRows).
type errRow struct {
	err error
}

func (r *errRow) Scan(dest ...any) error {
	return r.err
}
