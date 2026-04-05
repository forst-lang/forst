package parser

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestIsCapitalCase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want bool
	}{
		{"Foo", true},
		{"A", true},
		{"foo", false},
	}
	for _, tt := range tests {
		if got := isCapitalCase(tt.in); got != tt.want {
			t.Fatalf("isCapitalCase(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestUnexpectedTokenMessage_format(t *testing.T) {
	t.Parallel()
	tok := ast.Token{
		Type:   ast.TokenIdentifier,
		Value:  "@",
		FileID: "t.ft",
		Line:   2,
		Column: 5,
	}
	msg := unexpectedTokenMessage(tok, "IDENT")
	if !strings.Contains(msg, "Parse error") || !strings.Contains(msg, "t.ft") || !strings.Contains(msg, "Unexpected") {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestParseErrorMessage_format(t *testing.T) {
	t.Parallel()
	tok := ast.Token{
		Type:   ast.TokenIdentifier,
		Value:  "x",
		FileID: "t.ft",
		Line:   1,
		Column: 1,
	}
	msg := parseErrorMessage(tok, "boom")
	if !strings.Contains(msg, "boom") || !strings.Contains(msg, "t.ft") {
		t.Fatalf("unexpected message: %q", msg)
	}
}
