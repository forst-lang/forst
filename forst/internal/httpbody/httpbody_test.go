package httpbody

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestReadAll_underLimit(t *testing.T) {
	data := []byte("hello")
	got, err := ReadAll(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("got %q want %q", got, data)
	}
}

func TestReadAll_exactLimit(t *testing.T) {
	data := []byte("abc")
	got, err := ReadAll(bytes.NewReader(data), 3)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("got %q want %q", got, data)
	}
}

func TestReadAll_overLimit(t *testing.T) {
	data := []byte("abcd")
	_, err := ReadAll(bytes.NewReader(data), 3)
	if err == nil {
		t.Fatal("expected error for over-limit body")
	}
	if !IsTooLarge(err) {
		t.Fatalf("IsTooLarge=false for %v", err)
	}
}

func TestReadAll_zeroMaxUsesDefault(t *testing.T) {
	// Default is 10MB; a small body should succeed.
	_, err := ReadAll(bytes.NewReader([]byte("x")), 0)
	if err != nil {
		t.Fatalf("ReadAll with max 0: %v", err)
	}
}

func TestLimitReader_overLimit(t *testing.T) {
	r := LimitReader(strings.NewReader("too long"), 3)
	_, err := io.ReadAll(r)
	if err == nil {
		t.Fatal("expected limit error")
	}
	if !IsTooLarge(err) {
		t.Fatalf("IsTooLarge=false for %v", err)
	}
}

func TestIsTooLarge_nil(t *testing.T) {
	if IsTooLarge(nil) {
		t.Fatal("nil should not be too large")
	}
}
