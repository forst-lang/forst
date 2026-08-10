package invokeserver

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

const invokeNonceBytes = 32

type nonceStore struct {
	mu      sync.Mutex
	entries map[string]time.Time
	ttl     time.Duration
}

func newNonceStore(ttl time.Duration) *nonceStore {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &nonceStore{
		entries: make(map[string]time.Time),
		ttl:     ttl,
	}
}

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

func (s *nonceStore) sweepExpired(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepExpiredLocked(now)
}

func (s *nonceStore) sweepExpiredLocked(now time.Time) {
	for nonce, expiresAt := range s.entries {
		if now.After(expiresAt) {
			delete(s.entries, nonce)
		}
	}
}
