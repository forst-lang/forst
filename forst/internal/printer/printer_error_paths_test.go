package printer

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

type bogusTopLevelNode struct{}

func (bogusTopLevelNode) Kind() ast.NodeKind { return "bogus" }
func (bogusTopLevelNode) String() string     { return "bogus" }

func TestPrint_defaultIndentWhenEmpty(t *testing.T) {
	t.Parallel()
	out, err := Print(Config{Indent: ""}, []ast.Node{
		ast.PackageNode{Ident: ast.Ident{ID: "main"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "package main") {
		t.Fatalf("out = %q", out)
	}
}

func TestFormatTypeDefNode_defaultIndentWhenEmpty(t *testing.T) {
	t.Parallel()
	out, err := FormatTypeDefNode(Config{Indent: ""}, ast.TypeDefNode{
		Ident: "T",
		Expr:  ast.TypeDefShapeExpr{Shape: ast.ShapeNode{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "type T") {
		t.Fatalf("out = %q", out)
	}
}

func TestFormatTypeGuardNode_defaultIndentWhenEmpty(t *testing.T) {
	t.Parallel()
	out, err := FormatTypeGuardNode(Config{Indent: ""}, ast.TypeGuardNode{
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "x"},
			Type:  ast.NewBuiltinType(ast.TypeInt),
		},
		Ident: ast.Identifier("Positive"),
		Body:  []ast.Node{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "is (x Int) Positive") {
		t.Fatalf("out = %q", out)
	}
}

func TestPrint_unsupportedTopLevelNode(t *testing.T) {
	t.Parallel()
	_, err := Print(DefaultConfig(), []ast.Node{bogusTopLevelNode{}})
	if err == nil || !strings.Contains(err.Error(), "unsupported top-level") {
		t.Fatalf("err = %v", err)
	}
}

func TestPrintImportGroup_mixedImportKinds(t *testing.T) {
	t.Parallel()
	alias := ast.Ident{ID: "json"}
	p := printer{cfg: DefaultConfig()}
	out := p.printImportGroup(ast.ImportGroupNode{
		Imports: []ast.ImportNode{
			{Path: "fmt"},
			{Alias: &alias, Path: "encoding/json"},
			{Path: "database/sql", SideEffectOnly: true},
		},
	})
	if !strings.Contains(out, `"fmt"`) || !strings.Contains(out, `json "encoding/json"`) || !strings.Contains(out, `_ "database/sql"`) {
		t.Fatalf("out = %q", out)
	}
}

func TestPrintTypeDef_errorPayloadAndEmptyAssertion(t *testing.T) {
	t.Parallel()
	p := printer{cfg: DefaultConfig()}

	errOut, err := p.printTypeDef(ast.TypeDefNode{
		Ident: "E",
		Expr: ast.TypeDefErrorExpr{
			Payload: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"msg": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	})
	if err != nil || !strings.Contains(errOut, "error E") {
		t.Fatalf("error typedef: out=%q err=%v", errOut, err)
	}

	_, err = p.printTypeDef(ast.TypeDefNode{
		Ident: "Bad",
		Expr:  ast.TypeDefAssertionExpr{Assertion: nil},
	})
	if err == nil || !strings.Contains(err.Error(), "empty type def assertion") {
		t.Fatalf("err = %v", err)
	}

	_, err = p.printTypeDefExpr(&ast.TypeDefAssertionExpr{Assertion: nil})
	if err == nil {
		t.Fatal("expected nil pointer assertion error")
	}
}

func TestFlattenTypeDefSameOpChain_nilPointerBinary(t *testing.T) {
	t.Parallel()
	op, members := flattenTypeDefSameOpChain((*ast.TypeDefBinaryExpr)(nil))
	if op != "" || members != nil {
		t.Fatalf("op=%q members=%v", op, members)
	}
	members = flattenTypeDefSameOp((*ast.TypeDefBinaryExpr)(nil), ast.TokenBitwiseOr)
	if len(members) != 1 {
		t.Fatalf("members = %v", members)
	}
}

func TestMaybeMultilineTypeDefAlias_longUnionBreaksLines(t *testing.T) {
	t.Parallel()
	longName := strings.Repeat("X", 40)
	expr := ast.TypeDefBinaryExpr{
		Op: ast.TokenBitwiseOr,
		Left: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"firstFieldName": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
		Right: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"secondFieldName": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	}
	p := printer{cfg: Config{Indent: "  ", TypeDefLineWidth: 10}}
	out, err := p.printTypeDef(ast.TypeDefNode{Ident: ast.TypeIdent(longName), Expr: expr})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\n  |") {
		t.Fatalf("expected multiline union, got %q", out)
	}
}

func TestPrintExprFromNode_nilAndNonExpression(t *testing.T) {
	t.Parallel()
	p := printer{cfg: DefaultConfig()}
	if _, err := p.printExprFromNode(nil); err == nil {
		t.Fatal("expected nil node error")
	}
	if _, err := p.printExprFromNode(ast.PackageNode{Ident: ast.Ident{ID: "main"}}); err == nil {
		t.Fatal("expected non-expression error")
	}
}

func TestPrefixHelpers_emptyInput(t *testing.T) {
	t.Parallel()
	if got := prefixEachLine("\t", ""); got != "" {
		t.Fatalf("prefixEachLine empty = %q", got)
	}
	p := printer{cfg: DefaultConfig()}
	if got := p.prefixFirstLineOnly(""); got != "" {
		t.Fatalf("prefixFirstLineOnly empty = %q", got)
	}
}

func TestPrintParam_unsupportedParam(t *testing.T) {
	t.Parallel()
	p := printer{cfg: DefaultConfig()}
	_, err := p.printParam(bogusParam{})
	if err == nil || !strings.Contains(err.Error(), "unsupported param") {
		t.Fatalf("err = %v", err)
	}
}

type bogusParam struct{}

func (bogusParam) Kind() ast.NodeKind  { return "bogusParam" }
func (bogusParam) String() string      { return "bogus" }
func (bogusParam) GetIdent() string    { return "bogus" }
func (bogusParam) GetType() ast.TypeNode { return ast.TypeNode{Ident: ast.TypeInt} }
