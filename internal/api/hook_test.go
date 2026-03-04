package api

import (
	"context"
	"testing"

	"github.com/devaldrete/dotbrain/internal/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type mockDBTX struct {
	queries []string
	args    [][]any
}

func (m *mockDBTX) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (m *mockDBTX) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

type mockRow struct{}

func (m mockRow) Scan(dest ...any) error {
	return nil
}

func (m *mockDBTX) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	m.queries = append(m.queries, query)
	m.args = append(m.args, args)
	return mockRow{}
}

func TestDBNodeHook_OnNodeStart_WritesToDB(t *testing.T) {
	mockDB := &mockDBTX{}
	queries := db.New(mockDB)

	runID := pgtype.UUID{Bytes: [16]byte{1, 2, 3}, Valid: true}
	hook := NewDBNodeHook(queries, runID)

	ctx := context.Background()
	input := map[string]any{"hello": "world"}
	hook.OnNodeStart(ctx, "node1", input)

	if len(mockDB.queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(mockDB.queries))
	}

	if mockDB.args[0][2] != "node1" { // NodeID is 3rd arg in createNodeExecution
		t.Errorf("expected nodeID node1, got %v", mockDB.args[0][2])
	}
	if mockDB.args[0][3] != "running" { // Status
		t.Errorf("expected status running, got %v", mockDB.args[0][3])
	}
	
	// Test OnNodeComplete
	hook.OnNodeComplete(ctx, "node1", map[string]any{"result": "ok"})
	
	if len(mockDB.queries) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(mockDB.queries))
	}
	
	if mockDB.args[1][1] != "completed" { // Status is 2nd arg in updateNodeExecutionStatus
		t.Errorf("expected status completed, got %v", mockDB.args[1][1])
	}

	// Test OnNodeFail
	hook.OnNodeStart(ctx, "node2", nil)
	hook.OnNodeFail(ctx, "node2", context.DeadlineExceeded)

	if len(mockDB.queries) != 4 {
		t.Fatalf("expected 4 queries total, got %d", len(mockDB.queries))
	}

	if mockDB.args[3][1] != "failed" {
		t.Errorf("expected status failed, got %v", mockDB.args[3][1])
	}
}
