package printer

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

type unsupportedEnsureError struct{}

func (unsupportedEnsureError) String() string { return "unsupported" }

func TestShapeExprHasNestedFields(t *testing.T) {
	t.Parallel()

	p := printer{cfg: DefaultConfig()}
	tests := []struct {
		name string
		expr ast.ExpressionNode
		want bool
	}{
		{
			name: "plain shape without fields",
			expr: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}},
			want: false,
		},
		{
			name: "plain shape with fields",
			expr: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{"x": {}},
			},
			want: true,
		},
		{
			name: "reference to non-shape value",
			expr: ast.ReferenceNode{
				Value: ast.VariableNode{Ident: ast.Ident{ID: "v"}},
			},
			want: false,
		},
		{
			name: "reference to shape with fields",
			expr: ast.ReferenceNode{
				Value: ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{"x": {}},
				},
			},
			want: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := p.shapeExprHasNestedFields(tc.expr); got != tc.want {
				t.Fatalf("shapeExprHasNestedFields() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPrintImport_SideEffectOnly(t *testing.T) {
	t.Parallel()

	p := printer{cfg: DefaultConfig()}
	got := p.printImport(ast.ImportNode{
		Path:           "github.com/lib/pq",
		SideEffectOnly: true,
	})
	if got != `import _ "github.com/lib/pq"` {
		t.Fatalf("printImport() = %q", got)
	}
}

func TestPrintImport_AliasAndDefault(t *testing.T) {
	t.Parallel()

	p := printer{cfg: DefaultConfig()}
	alias := ast.Ident{ID: "json"}
	withAlias := p.printImport(ast.ImportNode{
		Alias: &alias,
		Path:  "encoding/json",
	})
	if withAlias != `import json "encoding/json"` {
		t.Fatalf("printImport alias = %q", withAlias)
	}

	defaultImport := p.printImport(ast.ImportNode{Path: "fmt"})
	if defaultImport != `import "fmt"` {
		t.Fatalf("printImport default = %q", defaultImport)
	}
}

func TestPrintStmt_LabeledBreakContinue(t *testing.T) {
	t.Parallel()

	p := printer{cfg: DefaultConfig()}
	label := &ast.Ident{ID: "outer"}

	breakOut, err := p.printStmt(&ast.BreakNode{Label: label})
	if err != nil {
		t.Fatalf("print break: %v", err)
	}
	if breakOut != "break outer" {
		t.Fatalf("break output = %q", breakOut)
	}

	continueOut, err := p.printStmt(&ast.ContinueNode{Label: label})
	if err != nil {
		t.Fatalf("print continue: %v", err)
	}
	if continueOut != "continue outer" {
		t.Fatalf("continue output = %q", continueOut)
	}
}

func TestPrintExpr_OkErrUnsupported(t *testing.T) {
	t.Parallel()

	p := printer{cfg: DefaultConfig()}
	for _, expr := range []ast.ExpressionNode{
		ast.OkExprNode{Value: ast.IntLiteralNode{Value: 1}},
		ast.ErrExprNode{Value: ast.StringLiteralNode{Value: "boom"}},
	} {
		_, err := p.printExpr(expr)
		if err == nil {
			t.Fatalf("expected unsupported expression error for %T", expr)
		}
	}
}

func TestPrintExpr_IncDecSuffixes(t *testing.T) {
	t.Parallel()

	p := printer{cfg: DefaultConfig()}
	inc, err := p.printExpr(ast.UnaryExpressionNode{
		Operator: ast.TokenPlusPlus,
		Operand:  ast.VariableNode{Ident: ast.Ident{ID: "i"}},
	})
	if err != nil {
		t.Fatalf("print ++ expression: %v", err)
	}
	if inc != "i++" {
		t.Fatalf("++ output = %q", inc)
	}

	dec, err := p.printExpr(ast.UnaryExpressionNode{
		Operator: ast.TokenMinusMinus,
		Operand:  ast.VariableNode{Ident: ast.Ident{ID: "i"}},
	})
	if err != nil {
		t.Fatalf("print -- expression: %v", err)
	}
	if dec != "i--" {
		t.Fatalf("-- output = %q", dec)
	}
}

func TestPrintEnsure_Branches(t *testing.T) {
	t.Parallel()

	p := printer{cfg: DefaultConfig()}
	errorType := ast.TypeError

	negated, err := p.printEnsure(ast.EnsureNode{
		Variable: ast.VariableNode{Ident: ast.Ident{ID: "err"}},
		Assertion: ast.AssertionNode{
			BaseType: &errorType,
			Constraints: []ast.ConstraintNode{
				{Name: "Nil"},
			},
		},
	})
	if err != nil {
		t.Fatalf("print negated ensure: %v", err)
	}
	if negated != "ensure !err" {
		t.Fatalf("negated ensure output = %q", negated)
	}

	var errNode ast.EnsureErrorNode = ast.EnsureErrorVar("boom")
	withError, err := p.printEnsure(ast.EnsureNode{
		Variable: ast.VariableNode{Ident: ast.Ident{ID: "x"}},
		Assertion: ast.AssertionNode{
			BaseType: &errorType,
			Constraints: []ast.ConstraintNode{
				{Name: "NotNil"},
			},
		},
		Error: &errNode,
		Block: &ast.EnsureBlockNode{},
	})
	if err != nil {
		t.Fatalf("print ensure with error var: %v", err)
	}
	if !strings.Contains(withError, "ensure x is Error.NotNil()") {
		t.Fatalf("missing ensure head in %q", withError)
	}
	if !strings.Contains(withError, "or boom") {
		t.Fatalf("missing ensure error var in %q", withError)
	}
	if strings.Contains(withError, "{\n") {
		t.Fatalf("empty ensure block should not render braces: %q", withError)
	}

	callErr := ast.EnsureErrorNode(ast.EnsureErrorCall{
		ErrorType: "TooSmall",
		ErrorArgs: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
	})
	callOut, err := p.printEnsure(ast.EnsureNode{
		Variable: ast.VariableNode{Ident: ast.Ident{ID: "x"}},
		Assertion: ast.AssertionNode{
			BaseType: &errorType,
			Constraints: []ast.ConstraintNode{
				{Name: "NotNil"},
			},
		},
		Error: &callErr,
	})
	if err != nil {
		t.Fatalf("print ensure with error call: %v", err)
	}
	if !strings.Contains(callOut, "or TooSmall(1)") {
		t.Fatalf("missing ensure error call in %q", callOut)
	}
}

func TestPrintEnsure_UnknownErrorType(t *testing.T) {
	t.Parallel()

	p := printer{cfg: DefaultConfig()}
	errorType := ast.TypeError
	unknown := ast.EnsureErrorNode(unsupportedEnsureError{})
	_, err := p.printEnsure(ast.EnsureNode{
		Variable: ast.VariableNode{Ident: ast.Ident{ID: "x"}},
		Assertion: ast.AssertionNode{
			BaseType: &errorType,
			Constraints: []ast.ConstraintNode{
				{Name: "NotNil"},
			},
		},
		Error: &unknown,
	})
	if err == nil {
		t.Fatal("expected unknown ensure error type to fail")
	}
}

func TestPrintWith_nonShapeWiring(t *testing.T) {
	t.Parallel()
	p := printer{cfg: DefaultConfig()}
	out, err := p.printWith(ast.WithNode{
		Wiring: ast.VariableNode{Ident: ast.Ident{ID: "wiring"}},
		Body:   []ast.Node{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "with wiring {") {
		t.Fatalf("out = %q", out)
	}
}

func TestPrintWith_multilineShapeWiring(t *testing.T) {
	t.Parallel()
	fields := make(map[string]ast.ShapeFieldNode)
	for i := range 20 {
		key := "fieldName" + string(rune('A'+i%26))
		fields[key] = ast.ShapeFieldNode{Type: &ast.TypeNode{Ident: ast.TypeString}}
	}
	p := printer{cfg: Config{Indent: "\t", TypeDefLineWidth: 20}}
	out, err := p.printWith(ast.WithNode{
		Wiring: ast.ShapeNode{Fields: fields},
		Body:   []ast.Node{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "with {") {
		t.Fatalf("out = %q", out)
	}
}

func TestPrintIf_withInit(t *testing.T) {
	t.Parallel()
	p := printer{cfg: DefaultConfig()}
	out, err := p.printIf(&ast.IfNode{
		Init: ast.AssignmentNode{
			LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "x"}}},
			RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
		},
		Condition: ast.BoolLiteralNode{Value: true},
		Body:      []ast.Node{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "x = 1; true") {
		t.Fatalf("out = %q", out)
	}
}

func TestPrintAssignment_branches(t *testing.T) {
	t.Parallel()
	p := printer{cfg: DefaultConfig()}

	short, err := p.printAssignment(ast.AssignmentNode{
		IsShort: true,
		LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "x"}}},
		RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
	})
	if err != nil || short != "x := 1" {
		t.Fatalf("short: %q err=%v", short, err)
	}

	intType := ast.NewBuiltinType(ast.TypeInt)
	varDecl, err := p.printAssignment(ast.AssignmentNode{
		LValues:       []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "n"}}},
		ExplicitTypes: []*ast.TypeNode{&intType},
	})
	if err != nil || varDecl != "var n: Int" {
		t.Fatalf("var decl: %q err=%v", varDecl, err)
	}

	compound, err := p.printAssignment(ast.AssignmentNode{
		LValues:    []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "n"}}},
		RValues:    []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
		CompoundOp: ast.TokenPlusEq,
	})
	if err != nil || compound != "n += 1" {
		t.Fatalf("compound: %q err=%v", compound, err)
	}

	varInit, err := p.printAssignment(ast.AssignmentNode{
		LValues:       []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "n"}}},
		RValues:       []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
		ExplicitTypes: []*ast.TypeNode{&intType},
	})
	if err != nil || varInit != "var n: Int = 1" {
		t.Fatalf("var init: %q err=%v", varInit, err)
	}

	deref, err := p.printAssignment(ast.AssignmentNode{
		LValues: []ast.ExpressionNode{ast.DereferenceNode{Value: ast.VariableNode{Ident: ast.Ident{ID: "p"}}}},
		RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 2}},
	})
	if err != nil || deref != "*p = 2" {
		t.Fatalf("deref assign: %q err=%v", deref, err)
	}
}

func TestPrintTypeDefExpr_binaryAndShape(t *testing.T) {
	t.Parallel()
	p := printer{cfg: DefaultConfig()}

	shapeOut, err := p.printTypeDefExpr(ast.TypeDefShapeExpr{
		Shape: ast.ShapeNode{
			Fields: map[string]ast.ShapeFieldNode{
				"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			},
		},
	})
	if err != nil || !strings.Contains(shapeOut, "{") {
		t.Fatalf("shape expr: %q err=%v", shapeOut, err)
	}

	andOut, err := p.printTypeDefExpr(ast.TypeDefBinaryExpr{
		Op:    ast.TokenBitwiseAnd,
		Left:  ast.TypeDefShapeExpr{Shape: ast.ShapeNode{}},
		Right: ast.TypeDefShapeExpr{Shape: ast.ShapeNode{}},
	})
	if err != nil || !strings.Contains(andOut, " & ") {
		t.Fatalf("and expr: %q err=%v", andOut, err)
	}

	errOut, err := p.printTypeDefExpr(ast.TypeDefErrorExpr{
		Payload: ast.ShapeNode{
			Fields: map[string]ast.ShapeFieldNode{
				"code": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			},
		},
	})
	if err != nil || !strings.HasPrefix(errOut, "error {") {
		t.Fatalf("error expr: %q err=%v", errOut, err)
	}
}

func TestPrintBinary_andCall(t *testing.T) {
	t.Parallel()
	p := printer{cfg: DefaultConfig()}
	out, err := p.printBinary(ast.BinaryExpressionNode{
		Left:     ast.IntLiteralNode{Value: 1},
		Operator: ast.TokenLogicalAnd,
		Right:    ast.BoolLiteralNode{Value: true},
	})
	if err != nil || out != "1 && true" {
		t.Fatalf("binary: %q err=%v", out, err)
	}
}

func TestPrintShapeFieldRHS_branches(t *testing.T) {
	t.Parallel()
	p := printer{cfg: DefaultConfig()}
	typeOnly, err := p.printShapeFieldRHS(ast.ShapeFieldNode{
		Type: &ast.TypeNode{Ident: ast.TypeInt},
	}, 0)
	if err != nil || typeOnly != "Int" {
		t.Fatalf("type only: %q err=%v", typeOnly, err)
	}
}

func TestPrintStmt_deferAndGo(t *testing.T) {
	t.Parallel()
	p := printer{cfg: DefaultConfig()}
	call := ast.FunctionCallNode{
		Function: ast.Ident{ID: "f"},
	}
	deferOut, err := p.printStmt(&ast.DeferNode{Call: call})
	if err != nil || deferOut != "defer f()" {
		t.Fatalf("defer: %q err=%v", deferOut, err)
	}
	goOut, err := p.printStmt(&ast.GoStmtNode{Call: call})
	if err != nil || goOut != "go f()" {
		t.Fatalf("go: %q err=%v", goOut, err)
	}
}

func TestEffectiveTypeDefLineWidth_default(t *testing.T) {
	t.Parallel()
	p := printer{cfg: Config{TypeDefLineWidth: 0}}
	if got := p.effectiveTypeDefLineWidth(); got != 80 {
		t.Fatalf("got %d", got)
	}
}

func TestPrintIf_elseIfAndElse(t *testing.T) {
	t.Parallel()
	p := printer{cfg: DefaultConfig()}
	out, err := p.printIf(&ast.IfNode{
		Condition: ast.BoolLiteralNode{Value: false},
		Body:      []ast.Node{},
		ElseIfs: []ast.ElseIfNode{{
			Condition: ast.BoolLiteralNode{Value: true},
			Body:      []ast.Node{},
		}},
		Else: &ast.ElseBlockNode{Body: []ast.Node{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "else if true") || !strings.Contains(out, "else {") {
		t.Fatalf("out = %q", out)
	}
}
