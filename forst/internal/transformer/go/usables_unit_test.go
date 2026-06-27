package transformergo

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
	gotoken "go/token"
)

func TestUsablesSlotSetKey_orderIndependent(t *testing.T) {
	a := usablesSlotSetKey([]typechecker.UsableSlot{
		{RootIdent: "Clock"},
		{RootIdent: "Logger"},
	})
	b := usablesSlotSetKey([]typechecker.UsableSlot{
		{RootIdent: "Logger"},
		{RootIdent: "Clock"},
	})
	if a != b {
		t.Fatalf("slot set keys differ: %q vs %q", a, b)
	}
}

func TestMergeWiringFrames_outerInnerAndShadow(t *testing.T) {
	outer := wiringFrame{
		fields: map[string]goast.Expr{
			"Logger": goast.NewIdent("outerLogger"),
			"Clock":  goast.NewIdent("outerClock"),
		},
	}
	inner := wiringFrame{
		fields: map[string]goast.Expr{
			"Clock": goast.NewIdent("innerClock"),
		},
	}
	merged := mergeWiringFrames(outer, inner)
	if merged.fields["Logger"] != outer.fields["Logger"] {
		t.Fatal("outer Logger should remain")
	}
	if merged.fields["Clock"] != inner.fields["Clock"] {
		t.Fatal("inner Clock should shadow outer")
	}
}

func TestTransformUseStatement_namedAndAnonymous(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tc.Defs["Logger"] = ast.TypeDefNode{
		Ident: "Logger",
		Expr:  ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}},
	}
	tr := setupTransformer(tc, log)

	named, err := tr.transformUseStatement(ast.UseNode{
		Ident:        &ast.Ident{ID: "logger"},
		ContractType: ast.TypeNode{Ident: "Logger"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assign, ok := named.(*goast.AssignStmt)
	if !ok {
		t.Fatalf("expected AssignStmt, got %T", named)
	}
	if assign.Tok != gotoken.DEFINE {
		t.Fatalf("tok = %v", assign.Tok)
	}
	sel, ok := assign.Rhs[0].(*goast.SelectorExpr)
	if !ok || sel.Sel.Name != "Logger" {
		t.Fatalf("rhs = %v", assign.Rhs[0])
	}

	anon, err := tr.transformUseStatement(ast.UseNode{
		ContractType: ast.TypeNode{Ident: "Logger"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := anon.(*goast.EmptyStmt); !ok {
		t.Fatalf("anonymous use should emit empty stmt, got %T", anon)
	}
}

func TestFunctionNeedsUsablesParam_skipsWiringRootAndReceiver(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	if tc.FunctionUsables == nil {
		tc.FunctionUsables = make(map[ast.Identifier][]typechecker.UsableSlot)
	}
	tc.FunctionUsables["expireToken"] = []typechecker.UsableSlot{{RootIdent: "Logger"}}
	tr := setupTransformer(tc, log)

	needs := tr.functionNeedsUsablesParam(ast.FunctionNode{
		Ident: ast.Ident{ID: "expireToken"},
	})
	if !needs {
		t.Fatal("expireToken should need usables param")
	}

	testFn := ast.FunctionNode{
		Ident: ast.Ident{ID: "TestExpireToken"},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: "t"},
				Type: ast.TypeNode{
					Ident:      ast.TypePointer,
					TypeParams: []ast.TypeNode{{Ident: "testing.T"}},
				},
			},
		},
	}
	if tr.functionNeedsUsablesParam(testFn) {
		t.Fatal("Test* wiring root should not get usables param")
	}

	method := ast.FunctionNode{
		Receiver: &ast.SimpleParamNode{Type: ast.TypeNode{Ident: "StdLogger"}},
		Ident:    ast.Ident{ID: "info"},
	}
	if tr.functionNeedsUsablesParam(method) {
		t.Fatal("receiver methods should not get usables param")
	}
}

func TestUsablesLowering_transitivePassThrough(t *testing.T) {
	src := `package main

type Logger = { info(msg String) }

func inner() {
	use logger: Logger
}

func outer() {
	inner()
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})

	if !containsIgnoreWhitespace(out, "func outer(usables Usables_") {
		t.Fatalf("outer should receive usables param, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "func inner(usables Usables_") {
		t.Fatalf("inner should receive usables param, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "inner(usables)") {
		t.Fatalf("outer should pass usables through to inner, got:\n%s", out)
	}
}
