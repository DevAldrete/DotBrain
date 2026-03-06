package api

import (
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// EventBus tests
// ---------------------------------------------------------------------------

func TestEventBus_SubscribeReceivesPublishedEvent(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Subscribe("run-1")
	defer bus.Unsubscribe("run-1", ch)

	event := RunEvent{Type: "run.started", RunID: "run-1"}
	bus.Publish("run-1", event)

	select {
	case got := <-ch:
		if got.Type != "run.started" {
			t.Errorf("expected run.started, got %q", got.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for event")
	}
}

func TestEventBus_MultipleSubscribersEachReceiveEvent(t *testing.T) {
	bus := NewEventBus()
	ch1 := bus.Subscribe("run-2")
	ch2 := bus.Subscribe("run-2")
	defer bus.Unsubscribe("run-2", ch1)
	defer bus.Unsubscribe("run-2", ch2)

	event := RunEvent{Type: "node.started", RunID: "run-2"}
	bus.Publish("run-2", event)

	for i, ch := range []<-chan RunEvent{ch1, ch2} {
		select {
		case got := <-ch:
			if got.Type != "node.started" {
				t.Errorf("subscriber %d: expected node.started, got %q", i, got.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("subscriber %d: timed out waiting for event", i)
		}
	}
}

func TestEventBus_UnsubscribeStopsReceivingEvents(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Subscribe("run-3")
	bus.Unsubscribe("run-3", ch)

	bus.Publish("run-3", RunEvent{Type: "run.completed", RunID: "run-3"})

	// Channel should receive nothing after unsubscribe (it will be closed or
	// simply not sent to). Either zero subscribers remain, so no send occurs.
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected no event after unsubscribe, but got one")
		}
		// closed channel is fine — that's acceptable behaviour
	case <-time.After(50 * time.Millisecond):
		// nothing received — correct
	}
}

func TestEventBus_PublishToUnknownRunIDIsNoop(t *testing.T) {
	bus := NewEventBus()
	// No subscribers for "run-99". This must not panic.
	bus.Publish("run-99", RunEvent{Type: "run.started", RunID: "run-99"})
}

func TestEventBus_ConcurrentPublishDoesNotRace(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Subscribe("run-4")
	defer bus.Unsubscribe("run-4", ch)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish("run-4", RunEvent{Type: "node.completed", RunID: "run-4"})
		}()
	}

	// Drain the channel concurrently so sends don't block
	go func() {
		for range ch {
		}
	}()

	wg.Wait()
}
