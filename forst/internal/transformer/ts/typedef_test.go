package transformerts

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func TestTransformTypeDef_errorNominalEmitsInterface(t *testing.T) {
	tr := New(typechecker.New(logrus.New(), false), nil)
	out, err := tr.transformTypeDef(ast.TypeDefNode{
		Ident: "NotPositive",
		Expr: ast.TypeDefErrorExpr{
			Payload: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"field": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "export interface NotPositive") {
		t.Fatalf("expected interface, got:\n%s", out)
	}
	if !strings.Contains(out, "field") || !strings.Contains(out, "string") {
		t.Fatalf("expected payload field in TS output:\n%s", out)
	}
}
