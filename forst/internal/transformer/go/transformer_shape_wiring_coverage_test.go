package transformergo

import (
	"strings"
	"testing"

	"forst/internal/ast"
	goast "go/ast"
)

func TestBuildWiringFrame_shapeAndCall(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tr := setupTransformer(setupTypeChecker(log), log)

	shapeFrame, err := tr.buildWiringFrame(ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"Logger": {Node: ast.StringLiteralNode{Value: "x"}},
		},
	})
	if err != nil {
		t.Fatalf("shape wiring: %v", err)
	}
	if len(shapeFrame.fields) != 1 {
		t.Fatalf("fields = %#v", shapeFrame.fields)
	}

	callFrame, err := tr.buildWiringFrame(ast.FunctionCallNode{
		Function: ast.Ident{ID: "services"},
	})
	if err != nil {
		t.Fatalf("call wiring: %v", err)
	}
	if callFrame.baseExpr == nil {
		t.Fatal("expected baseExpr from call wiring")
	}
}

func TestWiringFieldExpr_fieldOverrideAndBaseSelector(t *testing.T) {
	t.Parallel()
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	tr.wiringStack = []wiringFrame{
		{
			baseExpr: goast.NewIdent("svc"),
			fields: map[string]goast.Expr{
				"Clock": goast.NewIdent("clockImpl"),
			},
		},
	}
	if s := goExprString(t, tr.wiringFieldExpr("Clock")); s != "clockImpl" {
		t.Fatalf("field override: %s", s)
	}
	if s := goExprString(t, tr.wiringFieldExpr("Logger")); s != "svc.Logger" {
		t.Fatalf("base selector: %s", s)
	}
	tr.wiringStack = nil
	if tr.wiringFieldExpr("Missing") != nil {
		t.Fatal("expected nil for missing wiring")
	}
}

func TestTransformShapeNodeWithExpectedType_namedAndHashFallback(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tc.Defs["Point"] = ast.TypeDefNode{
		Ident: "Point",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
					"y": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	}
	tr := setupTransformer(tc, log)
	shape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"x": {Node: ast.IntLiteralNode{Value: 1}},
			"y": {Node: ast.IntLiteralNode{Value: 2}},
		},
	}
	named, err := tr.transformShapeNodeWithExpectedType(shape, &ast.TypeNode{Ident: "Point", TypeKind: ast.TypeKindUserDefined})
	if err != nil {
		t.Fatal(err)
	}
	if s := goExprString(t, named); !strings.Contains(s, `Point{`) {
		t.Fatalf("named literal: %s", s)
	}

	hash, err := tr.transformShapeNodeWithExpectedType(shape, &ast.TypeNode{Ident: "T_abc", TypeKind: ast.TypeKindHashBased})
	if err != nil {
		t.Fatal(err)
	}
	if s := goExprString(t, hash); !strings.Contains(s, `T_abc{`) {
		t.Fatalf("hash literal: %s", s)
	}
}

func TestFindExistingTypeForShape_baseTypeAndStructuralMatch(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tc.Defs["Pair"] = ast.TypeDefNode{
		Ident: "Pair",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"a": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
					"b": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	}
	tr := setupTransformer(tc, log)

	base := ast.TypeIdent("User")
	shape := &ast.ShapeNode{
		BaseType: &base,
		Fields:   map[string]ast.ShapeFieldNode{},
	}
	if got, ok := tr.findExistingTypeForShape(shape, nil); !ok || got != "User" {
		t.Fatalf("base type: got %q ok=%v", got, ok)
	}

	lit := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"a": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			"b": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
		},
	}
	if got, ok := tr.findExistingTypeForShape(lit, nil); !ok || got != "Pair" {
		t.Fatalf("structural match: got %q ok=%v", got, ok)
	}
}

func TestShapesMatch_comparesFieldTypes(t *testing.T) {
	t.Parallel()
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	a := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
		},
	}
	b := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
		},
	}
	c := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"x": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	if !tr.shapesMatch(a, b) {
		t.Fatal("expected match")
	}
	if tr.shapesMatch(a, c) {
		t.Fatal("expected mismatch")
	}
}

func TestCoerceGoStringAliasExprForType_assertionBaseString(t *testing.T) {
	t.Parallel()
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	expr := goast.NewIdent("slugVar")
	got := tr.coerceGoStringAliasExprForType(expr, ast.TypeNode{
		Ident: "Slug",
		Assertion: &ast.AssertionNode{
			BaseType: typeIdentPtr(string(ast.TypeString)),
		},
	})
	if s := goExprString(t, got); s != `string(slugVar)` {
		t.Fatalf("got %s", s)
	}
}

func TestTransformFunctionParamField_assertionBaseTypeOnly(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tc.Defs["User"] = ast.TypeDefNode{
		Ident: "User",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}
	tr := setupTransformer(tc, log)
	base := ast.TypeIdent("User")
	field, err := tr.transformFunctionParamField("u", ast.TypeNode{
		Assertion: &ast.AssertionNode{BaseType: &base},
	})
	if err != nil {
		t.Fatal(err)
	}
	if field.Type.(*goast.Ident).Name != "User" {
		t.Fatalf("field = %#v", field)
	}
}

func TestEnsureErrMissingMapKeyDecl_idempotent(t *testing.T) {
	var out TransformerOutput
	out.EnsureImport("errors")
	out.EnsureErrMissingMapKeyDecl()
	out.EnsureErrMissingMapKeyDecl()
	if len(out.valueDecls) != 1 {
		t.Fatalf("value decls: %d", len(out.valueDecls))
	}
}

func TestPipeline_withCallExpressionWiring(t *testing.T) {
	src := `package main

type Logger = { info(msg String) }
type NopLogger = {}

func (NopLogger) info(msg String) {}

func services(): { Logger: NopLogger } {
	return { Logger: NopLogger{} }
}

func work() {
	use logger: Logger
	logger.info("ok")
}

func main() {
	with services() {
		work()
	}
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})
	for _, sub := range []string{`services()`, `.Logger`} {
		if !strings.Contains(out, sub) {
			t.Fatalf("missing %q in:\n%s", sub, out)
		}
	}
	assertGoParses(t, out)
}
