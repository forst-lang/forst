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

type peerBackoffState struct {
	failures     int
	blockedUntil time.Time
}

type failedAuthLimiter struct {
	mu    sync.Mutex
	peers map[string]*peerBackoffState
}

func newFailedAuthLimiter() *failedAuthLimiter {
	return &failedAuthLimiter{peers: make(map[string]*peerBackoffState)}
}

func (l *failedAuthLimiter) Allow(peerKey string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	state := l.peers[peerKey]
	if state == nil {
		return true
	}
	return !now.Before(state.blockedUntil)
}

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

func (l *failedAuthLimiter) Reset(peerKey string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.peers, peerKey)
}
