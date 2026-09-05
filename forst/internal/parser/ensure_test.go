package parser

import (
	"forst/internal/ast"
	"strings"
	"testing"
)

func TestParseEnsure_bareBoolSuggestsIsTrue(t *testing.T) {
	t.Parallel()
	src := `package main

error Fail { msg: String }

func check(ok Bool): Result(String, Error) {
	ensure ok or Fail("no")
	return "ok"
}
`
	err := parseShouldFail(src)
	if err == nil {
		t.Fatal("expected parse error for bare ensure ok or …")
	}
	msg := err.Error()
	if !strings.Contains(msg, "ensure-missing-is") && !strings.Contains(msg, "ensure needs `is`") {
		t.Fatalf("expected ensure-missing-is diagnostic, got: %v", err)
	}
	if !strings.Contains(msg, "ensure ok is True()") {
		t.Fatalf("expected suggestion ensure ok is True(), got: %v", err)
	}
}

func TestParseEnsure_isTrueLiteralSuggestsTrueConstraint(t *testing.T) {
	t.Parallel()
	src := `package main

error Fail { msg: String }

func check(flag Bool): Result(String, Error) {
	ensure flag is true or Fail("no")
	return "ok"
}
`
	err := parseShouldFail(src)
	if err == nil {
		t.Fatal("expected parse error for ensure … is true")
	}
	msg := err.Error()
	if !strings.Contains(msg, "True()") {
		t.Fatalf("expected True() suggestion, got: %v", err)
	}
	if !strings.Contains(msg, "boolean literal") {
		t.Fatalf("expected boolean literal diagnostic, got: %v", err)
	}
}

func TestParseEnsure_okIsTrueOrNamedError(t *testing.T) {
	t.Parallel()
	src := `package main

error Fail { msg: String }

func check(ok Bool): Result(String, Error) {
	ensure ok is True() else Fail("no")
	return "ok"
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(nodes) < 2 {
		t.Fatalf("expected error typedef + function, got %d nodes", len(nodes))
	}
}

func TestParseEnsure_ElseFailureBlockInMain(t *testing.T) {
	t.Parallel()

	// 1. main accepts ensure !err else { ... }
	srcValid := `package main
func main() {
	err := false
	ensure !err else {
		println("failed")
	}
}
`
	_, err := NewTestParser(srcValid, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatalf("expected valid parse for ensure !err else { ... } in main, got: %v", err)
	}

	// 2. main rejects bare block ensure !err { ... }
	srcBareBlock := `package main
func main() {
	err := false
	ensure !err {
		println("failed")
	}
}
`
	errBare := parseShouldFail(srcBareBlock)
	if errBare == nil {
		t.Fatal("expected parse error for bare block in main")
	}
	if !strings.Contains(errBare.Error(), "ensure failure block requires 'else'") {
		t.Fatalf("expected 'ensure failure block requires else' message, got: %v", errBare)
	}

	// 3. main rejects typed else
	srcTypedElse := `package main
error Fail { msg: String }
func main() {
	err := false
	ensure !err else Fail("no")
}
`
	errTyped := parseShouldFail(srcTypedElse)
	if errTyped == nil {
		t.Fatal("expected parse error for typed else in main")
	}
	if !strings.Contains(errTyped.Error(), "typed failure in ensure statements is not allowed in main function") {
		t.Fatalf("expected typed failure in main error message, got: %v", errTyped)
	}
}

func TestParseEnsure_callSubjectRejected(t *testing.T) {
	t.Parallel()
	src := `package main

error Fail { msg: String }

func TransitionAllowed(from String, to String): Bool {
	return true
}

func check(from String, to String): Result(Bool, Error) {
	ensure TransitionAllowed(from, to) is True() else Fail("no")
	return true
}
`
	err := parseShouldFail(src)
	if err == nil {
		t.Fatal("expected parse error for ensure call subject")
	}
	msg := err.Error()
	if !strings.Contains(msg, "ensure subject must be an identifier") && !strings.Contains(msg, "refinement-non-place-subject") {
		t.Fatalf("expected identifier-only diagnostic, got: %v", err)
	}
}

func TestParseEnsure(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "ensure statement with type guard",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 10},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 12},
				{Type: ast.TokenEnsure, Value: "ensure", Line: 2, Column: 4},
				{Type: ast.TokenIdentifier, Value: "x", Line: 2, Column: 11},
				{Type: ast.TokenIs, Value: "is", Line: 2, Column: 13},
				{Type: ast.TokenIdentifier, Value: "String", Line: 2, Column: 16},
				{Type: ast.TokenRBrace, Value: "}", Line: 3, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 3, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				functionNode := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				if len(functionNode.Body) != 1 {
					t.Fatalf("Expected 1 statement in function body, got %d", len(functionNode.Body))
				}
				ensureNode := assertNodeType[ast.EnsureNode](t, functionNode.Body[0], "ast.EnsureNode")
				if ensureNode.Variable.Ident.ID == "" {
					t.Fatal("Expected ensure variable, got empty")
				}
				if !ensureNode.Variable.Ident.Span.IsSet() {
					t.Fatal("ensure subject Ident.Span must be set for per-occurrence inference and LSP hover")
				}
				wantSpan := ast.SpanFromToken(ast.Token{Line: 2, Column: 11, Value: "x"})
				if ensureNode.Variable.Ident.Span != wantSpan {
					t.Fatalf("ensure subject span: got %+v want %+v", ensureNode.Variable.Ident.Span, wantSpan)
				}
				if ensureNode.Assertion.BaseType == nil {
					t.Fatal("Expected ensure assertion with BaseType, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := ast.SetupTestLogger(nil)
			p := setupParser(tt.tokens, logger)
			nodes, err := p.ParseFile()
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}
			tt.validate(t, nodes)
		})
	}
}
