package reload

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestReloadCoordinator_drainWaitsForInFlight(t *testing.T) {
	t.Parallel()
	c := NewCoordinator()

	started := make(chan struct{})
	release := make(chan struct{})
	go c.TrackInFlight(func() {
		close(started)
		<-release
	})

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("in-flight work did not start")
	}

	drainDone := make(chan error, 1)
	go func() {
		drainDone <- c.BeginDrain(context.Background())
	}()

	select {
	case err := <-drainDone:
		t.Fatal("BeginDrain returned before in-flight finished:", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)

	select {
	case err := <-drainDone:
		if err != nil {
			t.Fatalf("BeginDrain: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("BeginDrain did not complete after in-flight finished")
	}

	if c.State() != StateRegenerating {
		t.Fatalf("state = %v, want regenerating", c.State())
	}
}

func TestReloadCoordinator_concurrentBeginDrain_singleFlight(t *testing.T) {
	t.Parallel()
	c := NewCoordinator()

	var drains atomic.Int32
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			errs <- c.BeginDrain(context.Background())
			drains.Add(1)
		}()
	}

	var first, second error
	select {
	case first = <-errs:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first BeginDrain")
	}
	select {
	case second = <-errs:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second BeginDrain")
	}

	if drains.Load() != 2 {
		t.Fatalf("drains = %d, want 2", drains.Load())
	}
	if (first == nil) == (second == nil) {
		t.Fatalf("expected exactly one successful drain, got first=%v second=%v", first, second)
	}
}
