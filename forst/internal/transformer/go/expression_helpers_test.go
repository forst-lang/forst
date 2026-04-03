package transformergo

import (
	goast "go/ast"
	"go/token"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestNegateDisjoinConjoin(t *testing.T) {
	x := &goast.Ident{Name: "x"}
	y := &goast.Ident{Name: "y"}

	n := negateCondition(x)
	if u, ok := n.(*goast.UnaryExpr); !ok || u.Op != token.NOT || u.X != x {
		t.Fatalf("negateCondition: %#v", n)
	}

	if d := disjoin(nil); d == nil {
		t.Fatal("disjoin nil")
	}
	if id, ok := disjoin(nil).(*goast.Ident); !ok || id.Name != BoolConstantFalse {
		t.Fatalf("disjoin empty: %#v", disjoin(nil))
	}

	d1 := disjoin([]goast.Expr{x})
	if id, ok := d1.(*goast.Ident); !ok || id.Name != "x" {
		t.Fatalf("disjoin one: %#v", d1)
	}
	d2 := disjoin([]goast.Expr{x, y})
	be, ok := d2.(*goast.BinaryExpr)
	if !ok || be.Op != token.LOR || be.X != x || be.Y != y {
		t.Fatalf("disjoin two: %#v", d2)
	}

	if c := conjoin(nil); c == nil {
		t.Fatal("conjoin nil")
	}
	c1 := conjoin([]goast.Expr{x})
	if id, ok := c1.(*goast.Ident); !ok || id.Name != "x" {
		t.Fatalf("conjoin one: %#v", c1)
	}
	c2 := conjoin([]goast.Expr{x, y})
	if be, ok := c2.(*goast.BinaryExpr); !ok || be.Op != token.LAND {
		t.Fatalf("conjoin two: %#v", c2)
	}
}

func TestTransformOperator_mappingAndError(t *testing.T) {
	log := logrus.New()
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	tests := []struct {
		op   ast.TokenIdent
		want token.Token
	}{
		{ast.TokenPlus, token.ADD},
		{ast.TokenMinus, token.SUB},
		{ast.TokenStar, token.MUL},
		{ast.TokenDivide, token.QUO},
		{ast.TokenModulo, token.REM},
		{ast.TokenEquals, token.EQL},
		{ast.TokenNotEquals, token.NEQ},
		{ast.TokenGreater, token.GTR},
		{ast.TokenLess, token.LSS},
		{ast.TokenGreaterEqual, token.GEQ},
		{ast.TokenLessEqual, token.LEQ},
		{ast.TokenLogicalAnd, token.LAND},
		{ast.TokenLogicalOr, token.LOR},
		{ast.TokenLogicalNot, token.NOT},
	}
	for _, tt := range tests {
		got, err := tr.transformOperator(tt.op)
		if err != nil || got != tt.want {
			t.Fatalf("op %s: got %v err %v want %v", tt.op, got, err, tt.want)
		}
	}

	_, err := tr.transformOperator(ast.TokenIdent("___not_an_op___"))
	if err == nil {
		t.Fatal("expected error for unsupported operator")
	}
}
