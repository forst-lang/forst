package parser

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestParseError_Location_and_Error(t *testing.T) {
	ctx := &Context{FileID: "file.ft"}
	tok := ast.Token{Line: 3, Column: 5, Value: "x"}
	e := &ParseError{Token: tok, Context: ctx, Msg: "bad token"}
	loc := e.Location()
	if !strings.Contains(loc, "file.ft") || !strings.Contains(loc, "3") || !strings.Contains(loc, "5") {
		t.Fatalf("Location: %q", loc)
	}
	msg := e.Error()
	if !strings.Contains(msg, "bad token") || !strings.Contains(msg, "file.ft") {
		t.Fatalf("Error(): %q", msg)
	}
}

func TestParseError_Location_without_context(t *testing.T) {
	e := &ParseError{Token: ast.Token{Line: 1, Column: 2}, Msg: "oops"}
	loc := e.Location()
	if !strings.Contains(loc, "line 1") || !strings.Contains(loc, "column 2") {
		t.Fatalf("Location() = %q", loc)
	}
}
