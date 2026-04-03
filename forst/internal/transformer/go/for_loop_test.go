package transformergo

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"testing"

	forstast "forst/internal/ast"
)

func TestTransformForNode_infiniteAndRange(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	main := forstast.FunctionNode{
		Ident: forstast.Ident{ID: "main"},
		Body: []forstast.Node{
			forstast.AssignmentNode{
				LValues: []forstast.VariableNode{{Ident: forstast.Ident{ID: "xs"}}},
				RValues: []forstast.ExpressionNode{
					forstast.ArrayLiteralNode{
						Value: []forstast.LiteralNode{forstast.IntLiteralNode{Value: 1}},
						Type:  forstast.TypeNode{Ident: forstast.TypeImplicit},
					},
				},
				IsShort: true,
			},
			&forstast.ForNode{Body: []forstast.Node{&forstast.BreakNode{}}},
			&forstast.ForNode{
				IsRange:    true,
				RangeX:     forstast.VariableNode{Ident: forstast.Ident{ID: "xs"}},
				RangeKey:   &forstast.Ident{ID: "i"},
				RangeValue: &forstast.Ident{ID: "v"},
				RangeShort: true,
				Body:       []forstast.Node{},
			},
		},
	}
	if err := tc.CheckTypes([]forstast.Node{main}); err != nil {
		t.Fatal(err)
	}

	for _, stmt := range main.Body {
		switch n := stmt.(type) {
		case *forstast.ForNode:
			goStmt, err := tr.transformForNode(n)
			if err != nil {
				t.Fatalf("transformForNode: %v", err)
			}
			var buf bytes.Buffer
			if err := format.Node(&buf, token.NewFileSet(), goStmt); err != nil {
				t.Fatal(err)
			}
			s := buf.String()
			if n.IsRange {
				if !contains(s, "range") || !contains(s, "i") || !contains(s, "v") {
					t.Fatalf("expected range loop, got %q", s)
				}
			} else {
				if !contains(s, "break") {
					t.Fatalf("expected break in body, got %q", s)
				}
			}
		}
	}
}

func TestTransformForNode_classicThreeClause(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	main := forstast.FunctionNode{
		Ident: forstast.Ident{ID: "main"},
		Body: []forstast.Node{
			&forstast.ForNode{
				Init: forstast.AssignmentNode{
					LValues: []forstast.VariableNode{{Ident: forstast.Ident{ID: "i"}}},
					RValues: []forstast.ExpressionNode{forstast.IntLiteralNode{Value: 0}},
					IsShort: true,
				},
				Cond: forstast.BinaryExpressionNode{
					Left:     forstast.VariableNode{Ident: forstast.Ident{ID: "i"}},
					Operator: forstast.TokenLess,
					Right:    forstast.IntLiteralNode{Value: 1},
				},
				Post: forstast.UnaryExpressionNode{
					Operator: forstast.TokenPlusPlus,
					Operand:  forstast.VariableNode{Ident: forstast.Ident{ID: "i"}},
				},
				Body: []forstast.Node{},
			},
		},
	}
	if err := tc.CheckTypes([]forstast.Node{main}); err != nil {
		t.Fatal(err)
	}
	loop := main.Body[0].(*forstast.ForNode)
	goStmt, err := tr.transformForNode(loop)
	if err != nil {
		t.Fatal(err)
	}
	fs, ok := goStmt.(*ast.ForStmt)
	if !ok || fs.Init == nil || fs.Cond == nil || fs.Post == nil {
		t.Fatalf("expected Go ForStmt with init/cond/post, got %#v", goStmt)
	}
}
