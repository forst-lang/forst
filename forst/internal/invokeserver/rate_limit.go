package invokeserver

import (
	"context"
)

type concurrencyLimiter struct {
	sem chan struct{}
}

func newConcurrencyLimiter(n int) *concurrencyLimiter {
	if n <= 0 {
		n = 64
	}
	return &concurrencyLimiter{sem: make(chan struct{}, n)}
}

func (l *concurrencyLimiter) Acquire(ctx context.Context) (release func(), err error) {
	select {
	case l.sem <- struct{}{}:
		return func() { <-l.sem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
