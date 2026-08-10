// auth_state holds the single live invoke secret and its monotonic generation counter.
package invokeserver

import (
	"crypto/rand"
	"fmt"
	"sync"
)

// invokeTokenBytes is the length of a freshly generated invoke HMAC key.
const invokeTokenBytes = 32

// authState stores one generation-bound token under a mutex; reload rotates in place.
type authState struct {
	mu         sync.RWMutex
	generation uint64
	token      []byte
}

// newAuthState returns empty auth state; call initToken or InstallAuth before use.
func newAuthState() *authState {
	return &authState{}
}

// generateToken reads invokeTokenBytes from crypto/rand.
func generateToken() ([]byte, error) {
	buf := make([]byte, invokeTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("generate invoke token: %w", err)
	}
	return buf, nil
}

// initToken generates the first token and bumps generation via rotate.
func (a *authState) initToken() error {
	token, err := generateToken()
	if err != nil {
		return err
	}
	a.rotate(token)
	return nil
}

// install replaces the live token and sets generation. When generation is zero,
// generation is incremented like rotate; otherwise generation is pinned.
func (a *authState) install(generation uint64, newToken []byte) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.token {
		a.token[i] = 0
	}
	a.token = append([]byte(nil), newToken...)
	if generation == 0 {
		a.generation++
	} else {
		a.generation = generation
	}
}

// rotate wipes the previous token buffer, installs newToken, increments generation,
// and returns the new generation.
func (a *authState) rotate(newToken []byte) uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.token {
		a.token[i] = 0
	}
	a.token = append([]byte(nil), newToken...)
	a.generation++
	return a.generation
}

// snapshot returns a copy of the current generation and token under read lock.
func (a *authState) snapshot() (generation uint64, token []byte) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.generation, append([]byte(nil), a.token...)
}

// currentGeneration returns the live generation without copying the token.
func (a *authState) currentGeneration() uint64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.generation
}
