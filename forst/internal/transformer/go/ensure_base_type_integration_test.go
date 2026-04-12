package transformergo

import (
	"bytes"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/typechecker"
)

// TestGetEnsureBaseType_inferredFromVariable covers the LookupEnsureBaseType path when
// Assertion.BaseType is nil (inferred from the subject variable after typecheck).
func TestGetEnsureBaseType_inferredFromVariable(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	s := "ab"
	ensure s is Min(1)
}
`
	log := ast.SetupTestLogger(nil)
	if !testing.Verbose() {
		log.SetOutput(bytes.NewBuffer(nil))
	}
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := typechecker.New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	tr := New(tc, log)

	var ensureStmt ast.EnsureNode
	var foundEnsure bool
	for _, n := range nodes {
		fn, ok := n.(ast.FunctionNode)
		if !ok || string(fn.Ident.ID) != "main" {
			continue
		}
		for _, st := range fn.Body {
			if en, ok := st.(ast.EnsureNode); ok {
				ensureStmt = en
				foundEnsure = true
				break
			}
		}
		break
	}
	if !foundEnsure {
		t.Fatal("ensure statement not found")
	}

	if err := tr.restoreScope(ensureStmt); err != nil {
		t.Fatal(err)
	}
	got, err := tr.getEnsureBaseType(ensureStmt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Ident != ast.TypeString {
		t.Fatalf("got base type %v want string", got.Ident)
	}
}
