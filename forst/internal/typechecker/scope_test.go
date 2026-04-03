package typechecker

import (
	"io"
	"strings"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func testLogger(t *testing.T) *logrus.Logger {
	t.Helper()
	log := logrus.New()
	log.SetOutput(io.Discard)
	return log
}

func TestScope_IsFunction_and_IsTypeGuard(t *testing.T) {
	log := testLogger(t)

	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	fnNode := ast.Node(fn)
	fnScope := NewScope(nil, &fnNode, log)
	if !fnScope.IsFunction() || fnScope.IsTypeGuard() {
		t.Fatalf("IsFunction=%v IsTypeGuard=%v", fnScope.IsFunction(), fnScope.IsTypeGuard())
	}

	tg := ast.TypeGuardNode{
		Ident: "IsNum",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "x"},
			Type:  ast.TypeNode{Ident: ast.TypeInt},
		},
		Body: []ast.Node{},
	}
	tgNode := ast.Node(tg)
	tgScope := NewScope(nil, &tgNode, log)
	if !tgScope.IsTypeGuard() || tgScope.IsFunction() {
		t.Fatalf("IsTypeGuard=%v IsFunction=%v", tgScope.IsTypeGuard(), tgScope.IsFunction())
	}

	tgPtr := &ast.TypeGuardNode{
		Ident: "P",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "y"},
			Type:  ast.TypeNode{Ident: ast.TypeString},
		},
		Body: []ast.Node{},
	}
	ptrNode := ast.Node(tgPtr)
	ptrScope := NewScope(nil, &ptrNode, log)
	if !ptrScope.IsTypeGuard() {
		t.Fatal("expected *TypeGuardNode in scope node")
	}
}

func TestScope_IsFunction_IsTypeGuard_panics_when_node_nil(t *testing.T) {
	log := testLogger(t)
	s := NewScope(nil, nil, log)

	t.Run("IsFunction", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()
		s.IsFunction()
	})

	t.Run("IsTypeGuard", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()
		s.IsTypeGuard()
	})
}

func TestScope_LookupVariable_parameter_resolves_in_function_and_type_guard(t *testing.T) {
	log := testLogger(t)

	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	fnNode := ast.Node(fn)
	fnScope := NewScope(nil, &fnNode, log)
	fnScope.RegisterSymbol(ast.Identifier("p"), []ast.TypeNode{{Ident: ast.TypeInt}}, SymbolParameter)
	sym, ok := fnScope.LookupVariable(ast.Identifier("p"))
	if !ok || sym.Identifier != "p" || sym.Kind != SymbolParameter {
		t.Fatalf("function scope: ok=%v sym=%+v", ok, sym)
	}

	tg := ast.TypeGuardNode{
		Ident: "G",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "x"},
			Type:  ast.TypeNode{Ident: ast.TypeInt},
		},
		Body: []ast.Node{},
	}
	tgNode := ast.Node(tg)
	tgScope := NewScope(nil, &tgNode, log)
	tgScope.RegisterSymbol(ast.Identifier("x"), []ast.TypeNode{{Ident: ast.TypeInt}}, SymbolParameter)
	sym2, ok2 := tgScope.LookupVariable(ast.Identifier("x"))
	if !ok2 || sym2.Identifier != "x" {
		t.Fatalf("type guard: ok=%v sym=%+v", ok2, sym2)
	}
}

func TestScope_LookupVariableType(t *testing.T) {
	log := testLogger(t)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	n := ast.Node(fn)
	s := NewScope(nil, &n, log)
	s.RegisterSymbol(ast.Identifier("v"), []ast.TypeNode{{Ident: ast.TypeString}}, SymbolVariable)
	ty, ok := s.LookupVariableType(ast.Identifier("v"))
	if !ok || len(ty) != 1 || ty[0].Ident != ast.TypeString {
		t.Fatalf("got %v ok=%v", ty, ok)
	}
}

func TestScope_String_nonGlobal(t *testing.T) {
	log := testLogger(t)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "outer"}, Body: []ast.Node{}}
	parentNode := ast.Node(fn)
	parent := NewScope(nil, &parentNode, log)
	inner := ast.FunctionNode{Ident: ast.Ident{ID: "inner"}, Body: []ast.Node{}}
	innerNode := ast.Node(inner)
	child := NewScope(parent, &innerNode, log)
	s := child.String()
	if s == "GlobalScope" || !strings.Contains(s, "Function") {
		t.Fatalf("String() = %q", s)
	}
}

func TestScope_DefineType_LookupType(t *testing.T) {
	log := testLogger(t)
	root := NewScope(nil, nil, log)
	ty := ast.TypeNode{Ident: ast.TypeIdent("MyT")}
	root.DefineType(ast.Identifier("MyT"), ty)

	sym, ok := root.LookupType(ast.Identifier("MyT"))
	if !ok || sym.Kind != SymbolType || len(sym.Types) != 1 {
		t.Fatalf("lookup: ok=%v sym=%+v", ok, sym)
	}

	child := NewScope(root, nil, log)
	sym2, ok2 := child.LookupType(ast.Identifier("MyT"))
	if !ok2 || sym2.Identifier != "MyT" {
		t.Fatalf("child lookup: ok=%v sym=%+v", ok2, sym2)
	}

	if _, ok := child.LookupType(ast.Identifier("Missing")); ok {
		t.Fatal("expected missing type")
	}
}
