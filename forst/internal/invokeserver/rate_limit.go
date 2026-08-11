// rate_limit caps concurrent in-flight POST /invoke handlers.
package invokeserver

import (
	"context"
)

// concurrencyLimiter bounds parallel invoke executions with a buffered channel semaphore.
type concurrencyLimiter struct {
	sem chan struct{}
}

// newConcurrencyLimiter returns a limiter allowing at most n concurrent acquires (default 64).
func newConcurrencyLimiter(n int) *concurrencyLimiter {
	if n <= 0 {
		n = 64
	}
	return &concurrencyLimiter{sem: make(chan struct{}, n)}
}

// Acquire blocks until a slot is free or ctx is canceled; release must be called to free the slot.
func (l *concurrencyLimiter) Acquire(ctx context.Context) (release func(), err error) {
	select {
	case l.sem <- struct{}{}:
		return func() { <-l.sem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
