// nonce_store issues single-use invoke challenge nonces with TTL and lazy expiry.
package invokeserver

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// invokeNonceBytes is the entropy length for each issued challenge nonce.
const invokeNonceBytes = 32

// nonceStore maps issued nonces to expiry times; consume deletes on successful use.
type nonceStore struct {
	mu      sync.Mutex
	entries map[string]time.Time
	ttl     time.Duration
}

// newNonceStore returns a store with the given TTL (defaults to 30s when ttl <= 0).
func newNonceStore(ttl time.Duration) *nonceStore {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &nonceStore{
		entries: make(map[string]time.Time),
		ttl:     ttl,
	}
}

// issue generates a fresh nonce, stores it until now+ttl, and returns expiry.
func (s *nonceStore) issue(now time.Time) (nonce string, expiresAt time.Time, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepExpiredLocked(now)
	buf := make([]byte, invokeNonceBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", time.Time{}, fmt.Errorf("issue invoke nonce: %w", err)
	}
	nonce = base64.RawURLEncoding.EncodeToString(buf)
	expiresAt = now.Add(s.ttl)
	s.entries[nonce] = expiresAt
	return nonce, expiresAt, nil
}

// consume removes nonce if present and unexpired; returns false on replay or expiry.
func (s *nonceStore) consume(nonce string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepExpiredLocked(now)
	expiresAt, ok := s.entries[nonce]
	if !ok {
		return false
	}
	delete(s.entries, nonce)
	return !now.After(expiresAt)
}

// sweepExpired deletes all entries past expiry (public wrapper for tests).
func (s *nonceStore) sweepExpired(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepExpiredLocked(now)
}

// sweepExpiredLocked removes expired entries; caller must hold s.mu.
func (s *nonceStore) sweepExpiredLocked(now time.Time) {
	for nonce, expiresAt := range s.entries {
		if now.After(expiresAt) {
			delete(s.entries, nonce)
		}
	}
}
