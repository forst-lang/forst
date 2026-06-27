package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFile_usablesExample(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "..", "examples", "in", "rfc", "requirements", "usables.ft")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	p := NewTestParser(string(src), nil)
	if _, err := p.ParseFile(); err != nil {
		t.Fatalf("parse usables.ft: %v", err)
	}
}
