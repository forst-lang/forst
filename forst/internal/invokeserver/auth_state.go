package invokeserver

import (
	"crypto/rand"
	"fmt"
	"sync"
)

const invokeTokenBytes = 32

type authState struct {
	mu         sync.RWMutex
	generation uint64
	token      []byte
}

func newAuthState() *authState {
	return &authState{}
}

func generateToken() ([]byte, error) {
	buf := make([]byte, invokeTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("generate invoke token: %w", err)
	}
	return buf, nil
}

func (a *authState) initToken() error {
	token, err := generateToken()
	if err != nil {
		return err
	}
	a.rotate(token)
	return nil
}

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

func (a *authState) snapshot() (generation uint64, token []byte) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.generation, append([]byte(nil), a.token...)
}

func (a *authState) currentGeneration() uint64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.generation
}
