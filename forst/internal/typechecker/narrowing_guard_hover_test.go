package typechecker

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestNarrowingTypeGuards_storedForIfBranchSubject(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "narrow_guards.ft")
	const src = `package main

type Password = String

is (password Password) Strong {
  ensure password is Min(12)
}

func f(): String {
  password: Password = "123456789012"
  if password is Strong() {
    return password
  }
  return ""
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	lex := lexer.New([]byte(src), "f.ft", logrus.New())
	tokens := lex.Lex()
	psr := parser.New(tokens, "f.ft", logrus.New())
	nodes, err := psr.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(logrus.New(), false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	var ret ast.ReturnNode
	var found bool
	var findIf func([]ast.Node)
	findIf = func(body []ast.Node) {
		for _, st := range body {
			switch x := st.(type) {
			case ast.IfNode:
				for _, b := range x.Body {
					if r, ok := b.(ast.ReturnNode); ok {
						ret = r
						found = true
						return
					}
				}
			case *ast.IfNode:
				for _, b := range x.Body {
					if r, ok := b.(ast.ReturnNode); ok {
						ret = r
						found = true
						return
					}
				}
			}
		}
	}
	for _, n := range nodes {
		var fn ast.FunctionNode
		switch v := n.(type) {
		case ast.FunctionNode:
			fn = v
		case *ast.FunctionNode:
			fn = *v
		default:
			continue
		}
		if string(fn.Ident.ID) != "f" {
			continue
		}
		ifn, ok := fn.Body[1].(*ast.IfNode)
		if !ok {
			t.Fatalf("expected *IfNode at body[1], got %T", fn.Body[1])
		}
		if _, ok := ifn.Condition.(ast.BinaryExpressionNode); !ok {
			t.Fatalf("condition type %T", ifn.Condition)
		}
		findIf(fn.Body)
		break
	}
	if !found || len(ret.Values) == 0 {
		t.Fatal("expected return password in if branch")
	}
	vn, ok := ret.Values[0].(ast.VariableNode)
	if !ok {
		t.Fatalf("expected variable, got %T", ret.Values[0])
	}
	guards := tc.NarrowingTypeGuardsForVariableOccurrence(vn)
	if len(guards) != 1 || guards[0] != "Strong" {
		t.Fatalf("expected [Strong], got %v", guards)
	}
}

// TestTypeGuardBody_nestedIfEnsure_runsFullInference verifies that `if` / `ensure` inside a type
// guard definition use the same inference as function bodies (narrowing, merge at if-chain end,
// ensure validation).
func TestTypeGuardBody_nestedIfEnsure_runsFullInference(t *testing.T) {
	t.Parallel()
	const src = `package main

type Password = String

is (password Password) Strong {
  if password is Password {
    ensure password is Min(12)
  }
}

func f(): String {
  return ""
}
`
	lex := lexer.New([]byte(src), "guard_body.ft", logrus.New())
	tokens := lex.Lex()
	psr := parser.New(tokens, "guard_body.ft", logrus.New())
	nodes, err := psr.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(logrus.New(), false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}
