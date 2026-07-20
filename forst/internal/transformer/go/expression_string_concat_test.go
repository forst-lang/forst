package transformergo

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/testutil"
	"forst/internal/typechecker"
	goast "go/ast"
	"go/token"
)

func setupStringAliasTransformer(t *testing.T) *Transformer {
	t.Helper()
	src := `package main

type Slug = String

func main() {
	s: Slug = "hello"
	println("x-" + s)
}
`
	tc, _ := typechecker.MustTypecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
	return setupTransformer(tc, setupTestLogger(nil))
}

func TestCoerceGoStringPlusAlias_wrapsAliasOperand(t *testing.T) {
	tr := setupStringAliasTransformer(t)
	left := ast.VariableNode{Ident: ast.Ident{ID: "s"}}
	right := ast.StringLiteralNode{Value: "x-"}
	leftGo, rightGo := tr.coerceGoStringPlusAlias(
		left,
		right,
		goast.NewIdent("s"),
		&goast.BasicLit{Kind: token.STRING, Value: `"x-"`},
	)
	got := goExprString(t, leftGo)
	if got != `string(s)` {
		t.Fatalf("left = %q", got)
	}
	if s := goExprString(t, rightGo); s != `"x-"` {
		t.Fatalf("right = %q", s)
	}
}

func TestCoerceGoStringConcatOperands_noCoercionForPlainString(t *testing.T) {
	tr := setupStringAliasTransformer(t)
	left := ast.StringLiteralNode{Value: "a"}
	right := ast.StringLiteralNode{Value: "b"}
	leftGo := &goast.BasicLit{Kind: token.STRING, Value: `"a"`}
	rightGo := &goast.BasicLit{Kind: token.STRING, Value: `"b"`}
	gotLeft, gotRight := tr.coerceGoStringConcatOperands(left, right, leftGo, rightGo)
	if goExprString(t, gotLeft) != `"a"` || goExprString(t, gotRight) != `"b"` {
		t.Fatalf("left=%s right=%s", goExprString(t, gotLeft), goExprString(t, gotRight))
	}
}

func TestCoerceGoStringConcatOperands_wrapsAliasWhenOtherIsString(t *testing.T) {
	tr := setupStringAliasTransformer(t)
	left := ast.VariableNode{Ident: ast.Ident{ID: "s"}}
	right := ast.StringLiteralNode{Value: "x-"}
	gotLeft, _ := tr.coerceGoStringConcatOperands(
		left,
		right,
		goast.NewIdent("s"),
		&goast.BasicLit{Kind: token.STRING, Value: `"x-"`},
	)
	if !strings.Contains(goExprString(t, gotLeft), "string(") {
		t.Fatalf("got %q", goExprString(t, gotLeft))
	}
}
