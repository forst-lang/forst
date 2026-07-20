package reload

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestWaitReady_readyAndDegraded(t *testing.T) {
	t.Parallel()
	c := NewCoordinator()
	if err := c.WaitReady(context.Background()); err != nil {
		t.Fatalf("ready: %v", err)
	}

	degradedErr := errors.New("compile failed")
	c.SetState(StateDegraded, degradedErr)
	if err := c.WaitReady(context.Background()); !errors.Is(err, degradedErr) {
		t.Fatalf("degraded with err: got %v", err)
	}

	c.SetState(StateDegraded, nil)
	if err := c.WaitReady(context.Background()); err != nil {
		t.Fatalf("degraded without err: %v", err)
	}
}

func TestWaitReady_drainingAndRegenerating_returnsErrNotReady(t *testing.T) {
	t.Parallel()
	c := NewCoordinator()
	for _, state := range []State{StateDraining, StateRegenerating} {
		c.SetState(state, nil)
		err := c.WaitReady(context.Background())
		if !errors.Is(err, ErrNotReady) {
			t.Fatalf("state %v: got %v, want ErrNotReady", state, err)
		}
	}
}

func TestSetState_readyIncrementsGenerationAndClearsDegraded(t *testing.T) {
	t.Parallel()
	c := NewCoordinator()
	if gen := c.Generation(); gen != 0 {
		t.Fatalf("initial generation = %d, want 0", gen)
	}

	degradedErr := errors.New("compile failed")
	c.SetState(StateDegraded, degradedErr)
	c.SetState(StateReady, nil)
	if gen := c.Generation(); gen != 1 {
		t.Fatalf("generation after first ready = %d, want 1", gen)
	}
	if err := c.WaitReady(context.Background()); err != nil {
		t.Fatalf("degraded cleared: %v", err)
	}

	c.SetState(StateReady, nil)
	if gen := c.Generation(); gen != 2 {
		t.Fatalf("generation after second ready = %d, want 2", gen)
	}
}

func TestBeginDrain_fromDegraded_entersRegenerating(t *testing.T) {
	t.Parallel()
	c := NewCoordinator()
	c.SetState(StateDegraded, errors.New("last compile failed"))
	if err := c.BeginDrain(context.Background()); err != nil {
		t.Fatalf("BeginDrain: %v", err)
	}
	if c.State() != StateRegenerating {
		t.Fatalf("state = %v, want regenerating", c.State())
	}
}

func TestBeginDrain_contextCancel_restoresPriorState(t *testing.T) {
	t.Parallel()
	t.Run("fromReady", func(t *testing.T) {
		t.Parallel()
		c := NewCoordinator()
		release := make(chan struct{})
		go c.TrackInFlight(func() { <-release })

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := c.BeginDrain(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("BeginDrain: %v", err)
		}
		if c.State() != StateReady {
			t.Fatalf("state = %v, want ready after cancel", c.State())
		}
		close(release)
	})

	t.Run("fromDegraded", func(t *testing.T) {
		t.Parallel()
		c := NewCoordinator()
		degradedErr := errors.New("compile failed")
		c.SetState(StateDegraded, degradedErr)
		release := make(chan struct{})
		go c.TrackInFlight(func() { <-release })

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := c.BeginDrain(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("BeginDrain: %v", err)
		}
		if c.State() != StateDegraded {
			t.Fatalf("state = %v, want degraded after cancel", c.State())
		}
		if err := c.WaitReady(context.Background()); !errors.Is(err, degradedErr) {
			t.Fatalf("degraded err: got %v", err)
		}
		close(release)
	})
}

func TestBeginDrain_rejectsWhileAlreadyDraining(t *testing.T) {
	t.Parallel()
	c := NewCoordinator()
	c.SetState(StateDraining, nil)
	err := c.BeginDrain(context.Background())
	if err == nil {
		t.Fatal("expected error when already draining")
	}
	if !strings.Contains(err.Error(), "draining") {
		t.Fatalf("error = %q, want state draining in message", err.Error())
	}
}

func TestBeginDrain_rejectsWhileRegenerating(t *testing.T) {
	t.Parallel()
	c := NewCoordinator()
	c.SetState(StateRegenerating, nil)
	err := c.BeginDrain(context.Background())
	if err == nil {
		t.Fatal("expected error when regenerating")
	}
	if !strings.Contains(err.Error(), "regenerating") {
		t.Fatalf("error = %q, want state regenerating in message", err.Error())
	}
}

func TestState_String_coversAllPhases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		state State
		want  string
	}{
		{StateReady, "ready"},
		{StateDraining, "draining"},
		{StateRegenerating, "regenerating"},
		{StateDegraded, "degraded"},
		{State(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.state.String(); got != tc.want {
			t.Fatalf("State(%d).String() = %q, want %q", tc.state, got, tc.want)
		}
	}
}

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

	errs := make(chan error, 2)
	for range 2 {
		go func() {
			errs <- c.BeginDrain(context.Background())
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

	if (first == nil) == (second == nil) {
		t.Fatalf("expected exactly one successful drain, got first=%v second=%v", first, second)
	}
}
