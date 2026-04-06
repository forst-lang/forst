package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestFieldHoverMarkdown_moveState(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	src := `package main

type GameState = {
	status: String,
}

type MoveResponse = {
	state: GameState,
}

func f(): MoveResponse {
	return { state: { status: "" } }
}

func main(): GameState {
	move := f()
	return move.state
}
`
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	md, parent, ok := tc.FieldHoverMarkdown(ast.Identifier("move"), ast.SourceSpan{}, []string{"state"})
	if !ok {
		t.Fatal("expected field hover")
	}
	if !strings.Contains(md, "state") || !strings.Contains(md, "GameState") {
		t.Fatalf("markdown: %q", md)
	}
	if string(parent) != "MoveResponse" {
		t.Fatalf("parent type for doc: got %q want MoveResponse", parent)
	}
}

func TestParentTypeIdentForFieldPath_nested(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	src := `package main

type GameState = {
	status: String,
}

type MoveResponse = {
	state: GameState,
}

func f(): MoveResponse {
	return { state: { status: "" } }
}

func main(): String {
	move := f()
	return move.state.status
}
`
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	bt, ok := tc.InferredTypesForVariableIdentifier("move")
	if !ok || len(bt) != 1 {
		t.Fatalf("move type: ok=%v types=%v", ok, bt)
	}
	pid, ok := tc.ParentTypeIdentForFieldPath(bt[0], []string{"state", "status"})
	if !ok || string(pid) != "GameState" {
		t.Fatalf("parent for status: ok=%v pid=%q", ok, pid)
	}
}
