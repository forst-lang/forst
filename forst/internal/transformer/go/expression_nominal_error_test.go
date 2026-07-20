package transformergo

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
)

func setupNominalErrorTransformerOnly(t *testing.T) *Transformer {
	t.Helper()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tc.Defs["NotFound"] = ast.TypeDefNode{
		Ident: "NotFound",
		Expr: ast.TypeDefErrorExpr{
			Payload: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"id": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}
	return setupTransformer(tc, log)
}

func setupNominalErrorCall(t *testing.T) (*Transformer, ast.FunctionCallNode) {
	t.Helper()
	src := `package main

error NotFound {
	id: String
}

func main() {
	e := NotFound({ id: "x" })
	println(e.id)
}
`
	log := setupTestLogger(nil)
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := setupTypeChecker(log)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	mainFn := nodes[len(nodes)-1].(ast.FunctionNode)
	assign := mainFn.Body[0].(ast.AssignmentNode)
	call := assign.RValues[0].(ast.FunctionCallNode)
	return setupTransformer(tc, log), call
}

func TestTransformNominalErrorConstructorCall_validCompositeLiteral(t *testing.T) {
	tr, call := setupNominalErrorCall(t)
	expr, ok, err := tr.transformNominalErrorConstructorCall(call)
	if err != nil || !ok || expr == nil {
		t.Fatalf("ok=%v err=%v expr=%#v", ok, err, expr)
	}
	if s := goExprString(t, expr); !strings.Contains(s, "NotFound{") {
		t.Fatalf("got %q", s)
	}
}

func TestTransformNominalErrorConstructorCall_wrongArity(t *testing.T) {
	tr := setupNominalErrorTransformerOnly(t)
	_, ok, err := tr.transformNominalErrorConstructorCall(ast.FunctionCallNode{
		Function:  ast.Ident{ID: "NotFound"},
		Arguments: nil,
	})
	if err == nil || !ok || !strings.Contains(err.Error(), "expects one payload") {
		t.Fatalf("got ok=%v err=%v", ok, err)
	}
	_, ok, err = tr.transformNominalErrorConstructorCall(ast.FunctionCallNode{
		Function: ast.Ident{ID: "NotFound"},
		Arguments: []ast.ExpressionNode{
			ast.ShapeNode{},
			ast.ShapeNode{},
		},
	})
	if err == nil || !ok {
		t.Fatalf("got ok=%v err=%v", ok, err)
	}
}

func TestTransformNominalErrorConstructorCall_nonShapeArg(t *testing.T) {
	tr := setupNominalErrorTransformerOnly(t)
	_, ok, err := tr.transformNominalErrorConstructorCall(ast.FunctionCallNode{
		Function:  ast.Ident{ID: "NotFound"},
		Arguments: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
	})
	if err == nil || !ok || !strings.Contains(err.Error(), "shape literal") {
		t.Fatalf("got ok=%v err=%v", ok, err)
	}
}

func TestTransformNominalErrorConstructorCall_nonErrorTypedef(t *testing.T) {
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	tr.TypeChecker.Defs["Counter"] = ast.TypeDefNode{
		Ident: "Counter",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"n": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	}
	expr, ok, err := tr.transformNominalErrorConstructorCall(ast.FunctionCallNode{
		Function: ast.Ident{ID: "Counter"},
		Arguments: []ast.ExpressionNode{
			ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}},
		},
	})
	if err != nil || ok || expr != nil {
		t.Fatalf("ok=%v err=%v expr=%#v", ok, err, expr)
	}
}
