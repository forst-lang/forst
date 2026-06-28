package transformergo

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
	gotoken "go/token"
)

func TestProvidersSlotSetKey_orderIndependent(t *testing.T) {
	a := providersSlotSetKey([]typechecker.ProviderSlot{
		{RootIdent: "Clock"},
		{RootIdent: "Logger"},
	})
	b := providersSlotSetKey([]typechecker.ProviderSlot{
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

func TestUniqueProvidersParamName(t *testing.T) {
	if got := uniqueProvidersParamName(nil); got != "providers" {
		t.Fatalf("no occupied: got %q", got)
	}
	if got := uniqueProvidersParamName([]string{"providers"}); got != "providers_" {
		t.Fatalf("one collision: got %q", got)
	}
	if got := uniqueProvidersParamName([]string{"providers", "providers_"}); got != "providers__" {
		t.Fatalf("two collisions: got %q", got)
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
	tr.currentFnProvidersName = "providers_"

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
	if x, ok := sel.X.(*goast.Ident); !ok || x.Name != "providers_" {
		t.Fatalf("use should read from currentFnProvidersName, got X=%v", sel.X)
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

func TestFunctionNeedsProvidersParam_skipsWiringRootAndReceiver(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	if tc.FunctionProviders == nil {
		tc.FunctionProviders = make(map[ast.Identifier][]typechecker.ProviderSlot)
	}
	tc.FunctionProviders["expireToken"] = []typechecker.ProviderSlot{{RootIdent: "Logger"}}
	tr := setupTransformer(tc, log)

	needs := tr.functionNeedsProvidersParam(ast.FunctionNode{
		Ident: ast.Ident{ID: "expireToken"},
	})
	if !needs {
		t.Fatal("expireToken should need providers param")
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
	if tr.functionNeedsProvidersParam(testFn) {
		t.Fatal("Test* wiring root should not get providers param")
	}

	method := ast.FunctionNode{
		Receiver: &ast.SimpleParamNode{Type: ast.TypeNode{Ident: "StdLogger"}},
		Ident:    ast.Ident{ID: "info"},
	}
	if tr.functionNeedsProvidersParam(method) {
		t.Fatal("receiver methods should not get providers param")
	}
}

func TestProvidersLowering_transitivePassThrough(t *testing.T) {
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

	if !containsIgnoreWhitespace(out, "func outer(providers Providers_") {
		t.Fatalf("outer should receive providers param, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "func inner(providers Providers_") {
		t.Fatalf("inner should receive providers param, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "inner(providers)") {
		t.Fatalf("outer should pass providers through to inner, got:\n%s", out)
	}
}

func TestProvidersLowering_paramNameCollision(t *testing.T) {
	src := `package main

type Logger = { info(msg String) }

func handler(providers String) {
	use logger: Logger
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})

	if !containsIgnoreWhitespace(out, "func handler(providers_ Providers_") {
		t.Fatalf("providers struct param should be providers_, got:\n%s", out)
	}
	if !containsIgnoreWhitespace(out, "logger := providers_.Logger") {
		t.Fatalf("use should bind from providers_, got:\n%s", out)
	}
}
