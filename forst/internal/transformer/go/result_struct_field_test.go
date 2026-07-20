package transformergo

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/testutil"
	"forst/internal/typechecker"
)

func setupResultStructFieldFixture(t *testing.T) (*Transformer, *typechecker.TypeChecker) {
	t.Helper()
	src := `package main

type Wrap = {
	r: Result(Int, Error),
}

func okInt(): Result(Int, Error) {
	return 42
}

func main() {
	x := okInt()
	w := { r: x }
}
`
	tc, _ := typechecker.MustTypecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
	log := setupTestLogger(nil)
	return setupTransformer(tc, log), tc
}

func narrowedCompoundResultVariable(t *testing.T) (*Transformer, ast.VariableNode) {
	t.Helper()
	src := `package main

type Wrap = {
	r: Result(Int, Error),
}

func okInt(): Result(Int, Error) {
	return 42
}

func main() {
	x := okInt()
	w := { r: x }
	if w.r is Ok() {
		println(w.r)
	}
}
`
	log := setupTestLogger(nil)
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := typechecker.New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	mainFn, ok := nodes[len(nodes)-1].(ast.FunctionNode)
	if !ok {
		t.Fatal("expected main function")
	}
	ifNode, ok := mainFn.Body[2].(ast.IfNode)
	if !ok {
		ifPtr, ok := mainFn.Body[2].(*ast.IfNode)
		if !ok {
			t.Fatalf("body[2] = %T", mainFn.Body[2])
		}
		ifNode = *ifPtr
	}
	printlnCall := ifNode.Body[0].(ast.FunctionCallNode)
	vn := printlnCall.Arguments[0].(ast.VariableNode)
	tr := setupTransformer(tc, log)
	return tr, vn
}

func TestCompoundResultFieldStorageSelector_emitsWDotR(t *testing.T) {
	tr, _ := setupResultStructFieldFixture(t)
	vn := ast.VariableNode{Ident: ast.Ident{ID: "w.r"}}
	sel, ok := tr.compoundResultFieldStorageSelector(vn)
	if !ok {
		t.Fatal("expected compound result field selector")
	}
	got := goExprString(t, sel)
	if !strings.Contains(got, "w.r") {
		t.Fatalf("got %q", got)
	}
}

func TestAppendResultStoragePayloadSelector_okUsesDotV(t *testing.T) {
	tr, vn := narrowedCompoundResultVariable(t)
	base, ok := tr.compoundResultFieldStorageSelector(vn)
	if !ok {
		t.Fatal("expected base selector")
	}
	got := tr.appendResultStoragePayloadSelector(vn, base)
	s := goExprString(t, got)
	if !strings.Contains(s, ".V") {
		t.Fatalf("got %q", s)
	}
}

func TestGoResultErrIdentForCompoundField(t *testing.T) {
	tr, _ := setupResultStructFieldFixture(t)
	vn := ast.VariableNode{Ident: ast.Ident{ID: "w.r"}}
	got, err := tr.goResultErrIdentForCompoundField(vn)
	if err != nil {
		t.Fatal(err)
	}
	s := goExprString(t, got)
	if !strings.Contains(s, ".Err") {
		t.Fatalf("got %q", s)
	}
}

func TestCompoundVarDeclaresResultField_rejectsSimpleVariable(t *testing.T) {
	tr, _ := setupResultStructFieldFixture(t)
	if tr.compoundVarDeclaresResultField(ast.VariableNode{Ident: ast.Ident{ID: "w"}}) {
		t.Fatal("simple variable should not declare compound result field")
	}
}

func TestGoResultSuccessValueExprForOkDiscriminator_compoundField(t *testing.T) {
	tr, _ := setupResultStructFieldFixture(t)
	left := ast.VariableNode{Ident: ast.Ident{ID: "w.r"}}
	got, err := tr.goResultSuccessValueExprForOkDiscriminator(left)
	if err != nil {
		t.Fatal(err)
	}
	s := goExprString(t, got)
	if !strings.Contains(s, ".V") {
		t.Fatalf("got %q", s)
	}
}

func TestGoResultErrIdentForCompoundField_rejectsNonResultField(t *testing.T) {
	tr, _ := setupResultStructFieldFixture(t)
	_, err := tr.goResultErrIdentForCompoundField(ast.VariableNode{Ident: ast.Ident{ID: "w"}})
	if err == nil {
		t.Fatal("expected error for non-compound variable")
	}
}
