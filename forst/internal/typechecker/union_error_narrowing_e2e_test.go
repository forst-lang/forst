package typechecker

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

// End-to-end: examples/in/union_error_narrowing.ft — Result(Int, ErrKind) and `if x is Err(ParseError)`
// narrows x to ParseError in the branch so onlyParseError(x) typechecks.
func TestCheckTypes_unionErrorNarrowing_exampleFile(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "examples", "in", "union_error_narrowing.ft"))
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(root)
	if err != nil {
		t.Fatalf("read %s: %v", root, err)
	}

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	p := parser.NewTestParser(string(src), log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}

// Same program as a string — documents the narrowing contract without relying on filesystem layout.
func TestCheckTypes_unionErrorNarrowing_inline(t *testing.T) {
	t.Parallel()
	src := `package main

error ParseError { code: Int }
error IoError { path: String }
type ErrKind = ParseError | IoError

func onlyParseError(p ParseError) {}

func mk(): Result(Int, ErrKind) {
	return 0
}

func main() {
	x := mk()
	if x is Err(ParseError) {
		onlyParseError(x)
	}
}
`
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}
