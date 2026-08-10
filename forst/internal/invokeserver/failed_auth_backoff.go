// failed_auth_backoff rate-limits repeated invalid auth attempts per peer key.
package invokeserver

import (
	"sync"
	"time"
)

const (
	defaultAuthFailureThreshold = 5
	defaultAuthBackoffBase      = 100 * time.Millisecond
	defaultAuthBackoffMax       = 30 * time.Second
)

// peerBackoffState tracks consecutive failures and the earliest retry time for one peer.
type peerBackoffState struct {
	failures     int
	blockedUntil time.Time
}

// failedAuthLimiter applies exponential backoff per peer after auth failures.
type failedAuthLimiter struct {
	mu    sync.Mutex
	peers map[string]*peerBackoffState
}

// newFailedAuthLimiter returns a limiter with an empty peer map.
func newFailedAuthLimiter() *failedAuthLimiter {
	return &failedAuthLimiter{peers: make(map[string]*peerBackoffState)}
}

// Allow reports whether peerKey may attempt auth at now (not in an active backoff window).
func (l *failedAuthLimiter) Allow(peerKey string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	state := l.peers[peerKey]
	if state == nil {
		return true
	}
	return !now.Before(state.blockedUntil)
}

// RecordFailure increments failures for peerKey and extends blockedUntil with capped exponential backoff.
func (l *failedAuthLimiter) RecordFailure(peerKey string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	state := l.peers[peerKey]
	if state == nil {
		state = &peerBackoffState{}
		l.peers[peerKey] = state
	}
	state.failures++
	shift := state.failures - defaultAuthFailureThreshold
	if shift < 0 {
		return
	}
	if shift > 10 {
		shift = 10
	}
	backoff := defaultAuthBackoffBase << shift
	if backoff > defaultAuthBackoffMax {
		backoff = defaultAuthBackoffMax
	}
	state.blockedUntil = now.Add(backoff)
}

// Reset clears backoff state for peerKey after a successful auth.
func (l *failedAuthLimiter) Reset(peerKey string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.peers, peerKey)
}
