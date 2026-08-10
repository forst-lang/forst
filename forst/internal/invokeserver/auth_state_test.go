package invokeserver

import (
	"bytes"
	"testing"
)

func TestGenerateToken_returns32RandomBytes(t *testing.T) {
	a, err := generateToken()
	if err != nil {
		t.Fatal(err)
	}
	b, err := generateToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != invokeTokenBytes || len(b) != invokeTokenBytes {
		t.Fatalf("len = %d %d", len(a), len(b))
	}
	if bytes.Equal(a, b) {
		t.Fatal("expected distinct tokens")
	}
}

func TestAuthState_rotate_incrementsGeneration(t *testing.T) {
	state := newAuthState()
	if got := state.rotate([]byte("one")); got != 1 {
		t.Fatalf("generation = %d", got)
	}
	if got := state.rotate([]byte("two")); got != 2 {
		t.Fatalf("generation = %d", got)
	}
}

func TestAuthState_rotate_wipesPreviousTokenBuffer(t *testing.T) {
	state := newAuthState()
	first := []byte("secret-token-value")
	state.rotate(first)
	for i := range first {
		first[i] = 0
	}
	_, token := state.snapshot()
	if string(token) == string(first) {
		t.Fatal("expected rotated token")
	}
}

func TestAuthState_snapshot_returnsCopyNotAliasedSlice(t *testing.T) {
	state := newAuthState()
	state.rotate([]byte("abc"))
	_, token := state.snapshot()
	token[0] = 'z'
	_, again := state.snapshot()
	if again[0] == 'z' {
		t.Fatal("snapshot must return a copy")
	}
}
