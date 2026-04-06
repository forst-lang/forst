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
	md, parent, ok := tc.FieldHoverMarkdown(ast.Identifier("move"), ast.SourceSpan{}, []string{"state"}, ast.SourceSpan{})
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

func TestFieldHoverMarkdown_compoundFieldEnsureNarrowing(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	src := `package main

type GameState = {
	cells: []String,
}

is (g GameState) ValidBoard() {
	ensure g.cells is Min(9)
	ensure g.cells is Max(9)
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
	lines := strings.Split(src, "\n")
	var gTok, cellsTok ast.Token
	found := false
	for i := 2; i < len(toks); i++ {
		if toks[i].Value != "cells" || toks[i-1].Type != ast.TokenDot || toks[i-2].Value != "g" {
			continue
		}
		ln := toks[i].Line
		if ln < 1 || ln > len(lines) {
			continue
		}
		if !strings.Contains(lines[ln-1], "Max") {
			continue
		}
		gTok = toks[i-2]
		cellsTok = toks[i]
		found = true
		break
	}
	if !found {
		t.Fatal("could not find g.cells on second ensure line")
	}
	dotted := ast.SpanBetweenTokens(gTok, cellsTok)
	// Root span left unset so base type of g resolves like other field-hover tests; dottedSpan
	// still matches the ensure subject `g.cells` for narrowing display.
	md, _, ok := tc.FieldHoverMarkdown(ast.Identifier("g"), ast.SourceSpan{}, []string{"cells"}, dotted)
	if !ok {
		t.Fatal("expected field hover for g.cells")
	}
	if !strings.Contains(md, "Array(String)") {
		t.Fatalf("markdown should mention element type; got %q", md)
	}
	if !strings.Contains(md, "Min(9)") {
		t.Fatalf("markdown should include prior ensure predicate Min(9); got %q", md)
	}
	if strings.Contains(md, "Max(9)") {
		t.Fatalf("markdown on second ensure subject must not include this line's Max(9); got %q", md)
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
