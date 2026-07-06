package transformergo

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/typechecker"
	goast "go/ast"
)

const loggedInGuardSrc = `package main

type AppContext = {
  sessionId: *String,
  user: *String,
}

is (ctx AppContext) LoggedIn() {
  ensure ctx.sessionId is Present()
  ensure ctx.user is Present()
}
`

func parseLoggedInGuard(t *testing.T) ([]ast.Node, *typechecker.TypeChecker, ast.Node, ast.TypeGuardNode) {
	t.Helper()
	log := ast.SetupTestLogger(nil)
	p := parser.NewTestParser(loggedInGuardSrc, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := typechecker.New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	var parseNode ast.Node
	var guard ast.TypeGuardNode
	for _, n := range nodes {
		switch g := n.(type) {
		case ast.TypeGuardNode:
			parseNode = n
			guard = g
		case *ast.TypeGuardNode:
			if g != nil {
				parseNode = n
				guard = *g
			}
		}
		if parseNode != nil {
			break
		}
	}
	if parseNode == nil {
		t.Fatal("LoggedIn guard not found in parse nodes")
	}
	return nodes, tc, parseNode, guard
}

func TestResolveTypeGuardScopeNode_defsHeapCopy(t *testing.T) {
	t.Parallel()
	nodes, tc, parseNode, guard := parseLoggedInGuard(t)

	if !tc.HasScopeForNode(parseNode) {
		t.Fatal("expected scope registered on parse node")
	}

	guardCopy := new(ast.TypeGuardNode)
	*guardCopy = guard
	tc.Defs[ast.TypeIdent("LoggedIn")] = guardCopy

	if tc.HasScopeForNode(ast.Node(*guardCopy)) {
		t.Fatal("Defs heap copy must not have registered scope")
	}
	if err := tc.RestoreScope(ast.Node(*guardCopy)); err == nil {
		t.Fatal("RestoreScope on Defs copy should fail")
	}

	tr := New(tc, nil)
	resolved := tr.resolveTypeGuardScopeNode(nodes, *guardCopy)
	if !tc.HasScopeForNode(resolved) {
		t.Fatalf("resolved node %T has no scope", resolved)
	}
	if err := tc.RestoreScope(resolved); err != nil {
		t.Fatalf("RestoreScope resolved: %v", err)
	}
}

func TestTransformForstFileToGo_loggedInFromDefsCopy(t *testing.T) {
	t.Parallel()
	nodes, tc, _, guard := parseLoggedInGuard(t)

	guardCopy := new(ast.TypeGuardNode)
	*guardCopy = guard
	tc.Defs[ast.TypeIdent("LoggedIn")] = guardCopy

	tr := New(tc, nil)
	goFile, err := tr.TransformForstFileToGo(nodes)
	if err != nil {
		t.Fatalf("TransformForstFileToGo: %v", err)
	}

	foundGuard := false
	for _, d := range goFile.Decls {
		fn, ok := d.(*goast.FuncDecl)
		if !ok {
			continue
		}
		if len(fn.Name.Name) >= 2 && fn.Name.Name[0] == 'G' && fn.Name.Name[1] == '_' {
			foundGuard = true
			break
		}
	}
	if !foundGuard {
		t.Fatal("expected emitted G_ type guard function for LoggedIn")
	}
}
