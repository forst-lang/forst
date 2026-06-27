package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFile_providersExample(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "..", "examples", "in", "rfc", "requirements", "providers.ft")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	p := NewTestParser(string(src), nil)
	if _, err := p.ParseFile(); err != nil {
		t.Fatalf("parse providers.ft: %v", err)
	}
}
