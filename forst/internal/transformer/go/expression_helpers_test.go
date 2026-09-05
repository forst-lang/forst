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
		t.Fatalf("negateCondition ident: %#v", n)
	}

	eq := &goast.BinaryExpr{X: x, Op: token.EQL, Y: goast.NewIdent("nil")}
	neq := negateCondition(eq)
	if be, ok := neq.(*goast.BinaryExpr); !ok || be.Op != token.NEQ || be.X != x {
		t.Fatalf("negate == : %#v", neq)
	}
	if be, ok := negateCondition(neq).(*goast.BinaryExpr); !ok || be.Op != token.EQL {
		t.Fatalf("negate != : %#v", negateCondition(neq))
	}

	lt := &goast.BinaryExpr{X: x, Op: token.LSS, Y: y}
	if be, ok := negateCondition(lt).(*goast.BinaryExpr); !ok || be.Op != token.GEQ {
		t.Fatalf("negate < : %#v", negateCondition(lt))
	}
	gt := &goast.BinaryExpr{X: x, Op: token.GTR, Y: y}
	if be, ok := negateCondition(gt).(*goast.BinaryExpr); !ok || be.Op != token.LEQ {
		t.Fatalf("negate > : %#v", negateCondition(gt))
	}
	le := &goast.BinaryExpr{X: x, Op: token.LEQ, Y: y}
	if be, ok := negateCondition(le).(*goast.BinaryExpr); !ok || be.Op != token.GTR {
		t.Fatalf("negate <= : %#v", negateCondition(le))
	}
	ge := &goast.BinaryExpr{X: x, Op: token.GEQ, Y: y}
	if be, ok := negateCondition(ge).(*goast.BinaryExpr); !ok || be.Op != token.LSS {
		t.Fatalf("negate >= : %#v", negateCondition(ge))
	}

	and := &goast.BinaryExpr{X: eq, Op: token.LAND, Y: lt}
	orNeg := negateCondition(and)
	be, ok := orNeg.(*goast.BinaryExpr)
	if !ok || be.Op != token.LOR {
		t.Fatalf("negate && : %#v", orNeg)
	}
	if l, ok := be.X.(*goast.BinaryExpr); !ok || l.Op != token.NEQ {
		t.Fatalf("De Morgan left: %#v", be.X)
	}
	if r, ok := be.Y.(*goast.BinaryExpr); !ok || r.Op != token.GEQ {
		t.Fatalf("De Morgan right: %#v", be.Y)
	}

	notX := &goast.UnaryExpr{Op: token.NOT, X: x}
	if negateCondition(notX) != x {
		t.Fatalf("cancel NOT: %#v", negateCondition(notX))
	}
	if got := negateCondition(&goast.ParenExpr{X: eq}); got.(*goast.BinaryExpr).Op != token.NEQ {
		t.Fatalf("paren unwrap: %#v", got)
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
	be, ok = d2.(*goast.BinaryExpr)
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
		{ast.TokenBitwiseAnd, token.AND},
		{ast.TokenBitwiseOr, token.OR},
		{ast.TokenXor, token.XOR},
		{ast.TokenLShift, token.SHL},
		{ast.TokenRShift, token.SHR},
		{ast.TokenAndNot, token.AND_NOT},
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
