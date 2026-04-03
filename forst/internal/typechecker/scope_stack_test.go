package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/hasher"
)

func TestNewScopeStack_initializes_global_and_current(t *testing.T) {
	t.Parallel()
	log := testLogger(t)
	h := hasher.New()
	ss := NewScopeStack(h, log)
	if ss == nil {
		t.Fatal("NewScopeStack returned nil")
	}
	g := ss.globalScope()
	if g == nil || !g.IsGlobal() {
		t.Fatalf("globalScope: %+v", g)
	}
	if ss.currentScope() != g {
		t.Fatal("current should start at global")
	}
}

func TestScopeStack_push_pop_and_globalScope(t *testing.T) {
	t.Parallel()
	log := testLogger(t)
	ss := NewScopeStack(hasher.New(), log)
	global := ss.globalScope()

	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "outer"},
		Body:  []ast.Node{ast.VariableNode{Ident: ast.Ident{ID: "o"}}},
	}
	outerNode := ast.Node(fn)
	outerScope := ss.pushScope(outerNode)
	if outerScope == nil || outerScope.Parent != global {
		t.Fatalf("outer scope: %+v", outerScope)
	}
	if ss.currentScope() != outerScope {
		t.Fatal("current should be outer after push")
	}

	inner := ast.FunctionNode{
		Ident: ast.Ident{ID: "inner"},
		Body:  []ast.Node{ast.VariableNode{Ident: ast.Ident{ID: "i"}}},
	}
	innerNode := ast.Node(inner)
	innerScope := ss.pushScope(innerNode)
	if innerScope.Parent != outerScope {
		t.Fatalf("inner parent: %+v", innerScope.Parent)
	}

	if ss.globalScope() != global {
		t.Fatal("globalScope should remain root")
	}

	ss.popScope()
	if ss.currentScope() != outerScope {
		t.Fatal("pop should return to outer")
	}
	ss.popScope()
	if ss.currentScope() != global {
		t.Fatal("pop should return to global")
	}
	ss.popScope()
	if ss.currentScope() != global {
		t.Fatal("pop at global should no-op")
	}
}

func TestScopeStack_findScope_and_restoreScope(t *testing.T) {
	t.Parallel()
	log := testLogger(t)
	ss := NewScopeStack(hasher.New(), log)

	// FunctionNode hashes only the body, not Ident — distinct bodies avoid map collisions.
	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "f"},
		Body:  []ast.Node{ast.VariableNode{Ident: ast.Ident{ID: "a"}}},
	}
	node := ast.Node(fn)
	scope := ss.pushScope(node)
	scope.RegisterSymbol(ast.Identifier("x"), []ast.TypeNode{{Ident: ast.TypeInt}}, SymbolVariable)

	inner := ast.FunctionNode{
		Ident: ast.Ident{ID: "g"},
		Body:  []ast.Node{ast.VariableNode{Ident: ast.Ident{ID: "b"}}},
	}
	ss.pushScope(ast.Node(inner))

	if s, ok := ss.findScope(node); !ok || s != scope {
		t.Fatalf("findScope: ok=%v scope=%p want=%p", ok, s, scope)
	}

	if err := ss.restoreScope(node); err != nil {
		t.Fatal(err)
	}
	if ss.currentScope() != scope {
		t.Fatal("restoreScope should make pushed scope current")
	}

	bad := ast.FunctionNode{
		Ident: ast.Ident{ID: "missing"},
		Body:  []ast.Node{ast.VariableNode{Ident: ast.Ident{ID: "z"}}},
	}
	if err := ss.restoreScope(ast.Node(bad)); err == nil {
		t.Fatal("expected error for unknown node")
	}
}

func TestScopeStack_LookupVariableType_walks_parents(t *testing.T) {
	t.Parallel()
	log := testLogger(t)
	ss := NewScopeStack(hasher.New(), log)

	ss.globalScope().RegisterSymbol(ast.Identifier("g"), []ast.TypeNode{{Ident: ast.TypeString}}, SymbolVariable)

	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	ss.pushScope(ast.Node(fn))
	ss.currentScope().RegisterSymbol(ast.Identifier("local"), []ast.TypeNode{{Ident: ast.TypeBool}}, SymbolVariable)

	if ty, ok := ss.LookupVariableType(ast.Identifier("local")); !ok || len(ty) != 1 || ty[0].Ident != ast.TypeBool {
		t.Fatalf("local: %v ok=%v", ty, ok)
	}
	if ty, ok := ss.LookupVariableType(ast.Identifier("g")); !ok || len(ty) != 1 || ty[0].Ident != ast.TypeString {
		t.Fatalf("global symbol from inner: %v ok=%v", ty, ok)
	}
	if _, ok := ss.LookupVariableType(ast.Identifier("none")); ok {
		t.Fatal("expected missing")
	}
}
