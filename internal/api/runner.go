package api

import (
	"context"
	"sync"
)

// activeRunRegistry maintains a mapping from active run IDs to their
// cancel functions. This allows in-flight workflow executions to be
// stopped via context cancellation.
type activeRunRegistry struct {
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

// newActiveRunRegistry creates an initialized registry.
func newActiveRunRegistry() activeRunRegistry {
	return activeRunRegistry{
		cancels: make(map[string]context.CancelFunc),
	}
}

// register stores a cancel function for the given run ID.
func (r *activeRunRegistry) register(runID string, cancel context.CancelFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cancels[runID] = cancel
}

// cancel calls the cancel function for the given run ID and removes it
// from the registry. Returns true if the run was found and cancelled.
func (r *activeRunRegistry) cancel(runID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	cancel, ok := r.cancels[runID]
	if ok {
		cancel()
		delete(r.cancels, runID)
	}
	return ok
}

// deregister removes a run from the registry without cancelling it.
// Called when a run completes normally.
func (r *activeRunRegistry) deregister(runID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cancels, runID)
}

// count returns the number of active runs (for testing/monitoring).
func (r *activeRunRegistry) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.cancels)
}
