package transformergo

import (
	"strings"
	"testing"

	"forst/internal/ast"
	goast "go/ast"
)

func TestTransformFunctionParamField_userShapeType(t *testing.T) {
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
	field, err := tr.transformFunctionParamField("u", ast.TypeNode{Ident: "User", TypeKind: ast.TypeKindUserDefined})
	if err != nil {
		t.Fatal(err)
	}
	if field.Names[0].Name != "u" || field.Type.(*goast.Ident).Name != "User" {
		t.Fatalf("field = %#v", field)
	}
}

func TestTransformShapeFieldType_emitsReferencedUserType(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tc.Defs["Address"] = ast.TypeDefNode{
		Ident: "Address",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"city": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}
	tr := setupTransformer(tc, log)
	expr, err := tr.transformShapeFieldType(ast.ShapeFieldNode{
		Type: &ast.TypeNode{Ident: "Address", TypeKind: ast.TypeKindUserDefined},
	})
	if err != nil {
		t.Fatal(err)
	}
	if goExprString(t, *expr) != "Address" {
		t.Fatalf("got %s", goExprString(t, *expr))
	}
}

func TestFindExistingTypeForShape_matchesRegisteredDef(t *testing.T) {
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
			"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			"y": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
		},
	}
	got, ok := tr.findExistingTypeForShape(shape, nil)
	if !ok || got != "Point" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestPipeline_ensureWithOrFallback(t *testing.T) {
	src := `package main

import "errors"

func bad(msg String): Error {
	return errors.New(msg)
}

func f(row Int): Result(String, Error) {
	ensure row is GreaterThan(-1) or bad("too small")
	return "ok"
}

func main() {
	println("hi")
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})
	if !strings.Contains(out, `bad("too small")`) {
		t.Fatalf("expected ensure-or fallback emit:\n%s", out)
	}
	assertGoParses(t, out)
}
