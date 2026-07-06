package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestRestoreScope_ifElseBranches_matchSymbolScope(t *testing.T) {
	t.Parallel()
	src := []byte(`package demo

func main(): Void {
	n := 2
	if n > 10 {
		z := 1
		println(string(z))
	} else if n < 5 {
		y := 2
		println(string(y))
	} else {
		w := 3
		println(string(w))
	}
}
`)
	nodes := parseNodesForTest(t, src)
	tc := testTypeChecker(t)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	fn := findFunctionNode(t, nodes, "main")
	ifn, ok := fn.Body[1].(*ast.IfNode)
	if !ok {
		t.Fatalf("body[1] want *IfNode, got %T", fn.Body[1])
	}

	cases := []struct {
		name     string
		scopeKey ast.Node
		varName  string
	}{
		{"if then", ifn, "z"},
		{"else if", &ifn.ElseIfs[0], "y"},
		{"else", ifn.Else, "w"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := tc.RestoreScope(c.scopeKey); err != nil {
				t.Fatalf("RestoreScope: %v", err)
			}
			sym, ok := tc.CurrentScope().LookupVariable(ast.Identifier(c.varName))
			if !ok || sym.Scope == nil {
				t.Fatalf("lookup %s: ok=%v", c.varName, ok)
			}
			if err := tc.RestoreScope(c.scopeKey); err != nil {
				t.Fatalf("RestoreScope again: %v", err)
			}
			if tc.CurrentScope() != sym.Scope {
				t.Fatalf("RestoreScope scope pointer != symbol scope for %s", c.varName)
			}
		})
	}
}

func findFunctionNode(t *testing.T, nodes []ast.Node, name string) ast.FunctionNode {
	t.Helper()
	for _, n := range nodes {
		fn, ok := n.(ast.FunctionNode)
		if ok && fn.Ident.ID == ast.Identifier(name) {
			return fn
		}
	}
	t.Fatalf("function %q not found", name)
	return ast.FunctionNode{}
}
