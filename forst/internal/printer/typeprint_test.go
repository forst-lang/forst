package printer

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func ptrConstraintValue(v ast.ValueNode) *ast.ValueNode {
	p := ast.ValueNode(v)
	return &p
}

func TestPrintType_builtins(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		typ  ast.TypeNode
		want string
	}{
		{"int", ast.TypeNode{Ident: ast.TypeInt}, "Int"},
		{"string", ast.TypeNode{Ident: ast.TypeString}, "String"},
		{"void", ast.TypeNode{Ident: ast.TypeVoid}, "Void"},
		{"error", ast.TypeNode{Ident: ast.TypeError}, "Error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := printType(tt.typ); got != tt.want {
				t.Fatalf("printType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrintType_mapAndPointer(t *testing.T) {
	t.Parallel()
	intNode := ast.TypeNode{Ident: ast.TypeInt}
	strNode := ast.TypeNode{Ident: ast.TypeString}
	mapNode := ast.TypeNode{
		Ident:      ast.TypeMap,
		TypeParams: []ast.TypeNode{intNode, strNode},
	}
	if got := printType(mapNode); got != "Map[Int, String]" {
		t.Fatalf("got %q", got)
	}

	ptrNode := ast.TypeNode{
		Ident:      ast.TypePointer,
		TypeParams: []ast.TypeNode{strNode},
	}
	if got := printType(ptrNode); got != "*String" {
		t.Fatalf("got %q", got)
	}
}

func TestPrintType_arrayAndBool(t *testing.T) {
	t.Parallel()
	if got := printType(ast.TypeNode{Ident: ast.TypeArray}); got != "[]" {
		t.Fatalf("empty array: got %q", got)
	}
	arrInt := ast.TypeNode{
		Ident:      ast.TypeArray,
		TypeParams: []ast.TypeNode{{Ident: ast.TypeInt}},
	}
	if got := printType(arrInt); got != "[]Int" {
		t.Fatalf("got %q", got)
	}
	if got := printType(ast.TypeNode{Ident: ast.TypeBool}); got != "Bool" {
		t.Fatalf("got %q", got)
	}
}

func TestPrintType_implicitAndShapeKeyword(t *testing.T) {
	t.Parallel()
	if got := printType(ast.TypeNode{Ident: ast.TypeImplicit}); got != "" {
		t.Fatalf("implicit: got %q", got)
	}
	if got := printType(ast.TypeNode{Ident: ast.TypeShape}); got != "Shape" {
		t.Fatalf("got %q", got)
	}
}

func TestPrintType_userGeneric(t *testing.T) {
	t.Parallel()
	tn := ast.TypeNode{
		Ident: ast.TypeIdent("Box"),
		TypeParams: []ast.TypeNode{
			{Ident: ast.TypeInt},
			{Ident: ast.TypeString},
		},
	}
	if got := printType(tn); got != "Box<Int, String>" {
		t.Fatalf("got %q", got)
	}
}

func TestPrintType_userNominalIdent(t *testing.T) {
	t.Parallel()
	if got := printType(ast.TypeNode{Ident: ast.TypeIdent("NotPositive")}); got != "NotPositive" {
		t.Fatalf("got %q", got)
	}
}

func TestPrintType_result(t *testing.T) {
	t.Parallel()
	res := ast.TypeNode{
		Ident: ast.TypeResult,
		TypeParams: []ast.TypeNode{
			{Ident: ast.TypeString},
			{Ident: ast.TypeError},
		},
	}
	if got := printType(res); got != "Result(String, Error)" {
		t.Fatalf("Result: got %q", got)
	}
}

func TestPrintType_tuple(t *testing.T) {
	t.Parallel()
	tup := ast.TypeNode{
		Ident: ast.TypeTuple,
		TypeParams: []ast.TypeNode{
			{Ident: ast.TypeInt},
			{Ident: ast.TypeString},
		},
	}
	if got := printType(tup); got != "Tuple(Int, String)" {
		t.Fatalf("Tuple: got %q", got)
	}
}

func TestQuoteString_roundTrip(t *testing.T) {
	t.Parallel()
	s := "a\"b\nc"
	if got := quoteString(s); got != `"a\"b\nc"` {
		t.Fatalf("got %q", got)
	}
}

func TestPrintType_typeAssertion_and_namedWithConstraints(t *testing.T) {
	t.Parallel()
	base := ast.TypeString
	a := ast.AssertionNode{
		BaseType: &base,
		Constraints: []ast.ConstraintNode{
			{Name: "Min", Args: []ast.ConstraintArgumentNode{{Value: ptrConstraintValue(ast.IntLiteralNode{Value: 2})}}},
		},
	}
	if got := printType(ast.TypeNode{Ident: ast.TypeAssertion, Assertion: &a}); got != "String.Min(2)" {
		t.Fatalf("TypeAssertion: got %q", got)
	}

	tNamed := ast.TypeIdent("Row")
	a2 := ast.AssertionNode{
		Constraints: []ast.ConstraintNode{
			{
				Name: "Between",
				Args: []ast.ConstraintArgumentNode{
					{Value: ptrConstraintValue(ast.IntLiteralNode{Value: 1})},
					{Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	}
	got := printType(ast.TypeNode{Ident: tNamed, Assertion: &a2})
	if got != "Row.Between(1, Int)" {
		t.Fatalf("named + constraints: got %q", got)
	}
}

func TestPrintType_typeShape_withAssertion(t *testing.T) {
	t.Parallel()
	base := ast.TypeString
	a := ast.AssertionNode{
		BaseType:    &base,
		Constraints: []ast.ConstraintNode{{Name: "Present", Args: []ast.ConstraintArgumentNode{}}},
	}
	if got := printType(ast.TypeNode{Ident: ast.TypeShape, Assertion: &a}); !strings.Contains(got, "Present") {
		t.Fatalf("got %q", got)
	}
}

func TestFormatTypeGuardNode_roundTrip(t *testing.T) {
	t.Parallel()
	g := ast.TypeGuardNode{
		Ident: ast.Identifier("Positive"),
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "n"},
			Type:  ast.TypeNode{Ident: ast.TypeInt},
		},
		Body: []ast.Node{
			ast.ReturnNode{Values: []ast.ExpressionNode{ast.BoolLiteralNode{Value: true}}},
		},
	}
	out, err := FormatTypeGuardNode(DefaultConfig(), g)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "is (") || !strings.Contains(out, "Positive") || !strings.Contains(out, "n Int") {
		t.Fatalf("FormatTypeGuardNode: %q", out)
	}
}
