package reload

import (
	"context"
	"errors"
	"sync"
)

// ErrNotReady is returned when the server is draining or regenerating.
var ErrNotReady = errors.New("server not ready")

// State is the reload coordinator phase.
type State int

const (
	StateReady State = iota
	StateDraining
	StateRegenerating
	StateDegraded
)

// ReloadCoordinator tracks reload drain, in-flight invokes, and generation.
type ReloadCoordinator struct {
	mu         sync.Mutex
	cond       *sync.Cond
	state      State
	generation uint64
	degraded   error
	inFlight   sync.WaitGroup
}

// NewCoordinator returns a coordinator in the Ready state.
func NewCoordinator() *ReloadCoordinator {
	c := &ReloadCoordinator{state: StateReady}
	c.cond = sync.NewCond(&c.mu)
	return c
}

// Generation returns the current reload generation (incremented on Ready).
func (c *ReloadCoordinator) Generation() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.generation
}

// State returns the current coordinator phase.
func (c *ReloadCoordinator) State() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// BeginDrain enters Draining, waits for in-flight work, then moves to Regenerating.
func (c *ReloadCoordinator) BeginDrain(ctx context.Context) error {
	c.mu.Lock()
	if c.state != StateReady && c.state != StateDegraded {
		state := c.state
		c.mu.Unlock()
		return errors.New("cannot drain: state " + state.String())
	}
	c.state = StateDraining
	c.mu.Unlock()

	done := make(chan struct{})
	go func() {
		c.inFlight.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}

	c.mu.Lock()
	c.state = StateRegenerating
	c.cond.Broadcast()
	c.mu.Unlock()
	return nil
}

// WaitReady blocks until the coordinator is Ready or Degraded, or ctx is canceled.
// Returns ErrNotReady immediately while Draining or Regenerating.
func (c *ReloadCoordinator) WaitReady(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for {
		switch c.state {
		case StateReady:
			return nil
		case StateDegraded:
			if c.degraded != nil {
				return c.degraded
			}
			return nil
		case StateDraining, StateRegenerating:
			return ErrNotReady
		}
	}
}

// SetState updates the coordinator phase. Entering Ready bumps generation.
func (c *ReloadCoordinator) SetState(state State, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state
	if state == StateDegraded {
		c.degraded = err
	}
	if state == StateReady {
		c.generation++
		c.degraded = nil
	}
	c.cond.Broadcast()
}

// TrackInFlight runs fn while counted as in-flight for drain.
func (c *ReloadCoordinator) TrackInFlight(fn func()) {
	c.inFlight.Add(1)
	defer c.inFlight.Done()
	fn()
}

func (s State) String() string {
	switch s {
	case StateReady:
		return "ready"
	case StateDraining:
		return "draining"
	case StateRegenerating:
		return "regenerating"
	case StateDegraded:
		return "degraded"
	default:
		return "unknown"
	}
}
