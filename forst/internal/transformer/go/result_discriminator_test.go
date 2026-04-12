package transformergo

import (
	"bytes"
	"errors"
	"go/format"
	"go/token"
	"strings"
	"testing"

	"forst/internal/ast"
	goast "go/ast"
)

func TestExpressionFromConstraintArg_errors(t *testing.T) {
	t.Parallel()
	_, err := expressionFromConstraintArg(ast.ConstraintArgumentNode{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errMissingConstraintArgValue) {
		t.Fatalf("expected errMissingConstraintArgValue, got %v", err)
	}
}

func TestGoResultErrIdentForExpr_branches(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	_, err := tr.goResultErrIdentForExpr(ast.IntLiteralNode{Value: 1})
	if err == nil {
		t.Fatal("expected error for non-variable")
	}

	tr.resultLocalSplit = map[string]resultLocalSplit{"x": {errGoName: "xErr"}}
	got, err := tr.goResultErrIdentForExpr(ast.VariableNode{Ident: ast.Ident{ID: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	if id, ok := got.(*goast.Ident); !ok || id.Name != "xErr" {
		t.Fatalf("got %#v", got)
	}

	tr.resultLocalSplit = nil
	_, err = tr.goResultErrIdentForExpr(ast.VariableNode{Ident: ast.Ident{ID: "x"}})
	if err == nil || !strings.Contains(err.Error(), "missing local split") {
		t.Fatalf("unexpected: %v", err)
	}

	_, err = tr.goResultErrIdentForExpr(ast.VariableNode{Ident: ast.Ident{ID: "w.r"}})
	if err == nil || !strings.Contains(err.Error(), "not a lowered") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestTransformResultIsDiscriminator_splitLocal(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	tr.resultLocalSplit = map[string]resultLocalSplit{
		"x": {errGoName: "xErr", successGoNames: []string{"x"}},
	}
	left := ast.VariableNode{Ident: ast.Ident{ID: "x"}}

	t.Run("Ok_no_args", func(t *testing.T) {
		out, err := tr.transformResultIsDiscriminator(left, ast.ConstraintNode{Name: "Ok"})
		if err != nil {
			t.Fatal(err)
		}
		if s := exprString(t, out); !strings.Contains(s, "xErr") || !strings.Contains(s, "nil") {
			t.Fatalf("got %q", s)
		}
	})

	t.Run("Ok_with_int_arg", func(t *testing.T) {
		var v ast.ValueNode = ast.IntLiteralNode{Value: 42}
		arg := ast.ConstraintArgumentNode{Value: &v}
		out, err := tr.transformResultIsDiscriminator(left, ast.ConstraintNode{Name: "Ok", Args: []ast.ConstraintArgumentNode{arg}})
		if err != nil {
			t.Fatal(err)
		}
		s := exprString(t, out)
		if !strings.Contains(s, "xErr") || !strings.Contains(s, "42") {
			t.Fatalf("got %q", s)
		}
	})

	t.Run("Err_no_args", func(t *testing.T) {
		out, err := tr.transformResultIsDiscriminator(left, ast.ConstraintNode{Name: "Err"})
		if err != nil {
			t.Fatal(err)
		}
		if s := exprString(t, out); !strings.Contains(s, "!=") {
			t.Fatalf("got %q", s)
		}
	})

	t.Run("Err_with_var_arg", func(t *testing.T) {
		var v ast.ValueNode = ast.VariableNode{Ident: ast.Ident{ID: "e"}}
		arg := ast.ConstraintArgumentNode{Value: &v}
		out, err := tr.transformResultIsDiscriminator(left, ast.ConstraintNode{Name: "Err", Args: []ast.ConstraintArgumentNode{arg}})
		if err != nil {
			t.Fatal(err)
		}
		s := exprString(t, out)
		if !strings.Contains(s, "xErr") || !strings.Contains(s, "e") {
			t.Fatalf("got %q", s)
		}
	})

	t.Run("Err_with_type_arg", func(t *testing.T) {
		arg := ast.ConstraintArgumentNode{Type: &ast.TypeNode{Ident: ast.TypeInt}}
		out, err := tr.transformResultIsDiscriminator(left, ast.ConstraintNode{Name: "Err", Args: []ast.ConstraintArgumentNode{arg}})
		if err != nil {
			t.Fatal(err)
		}
		s := exprString(t, out)
		if !strings.Contains(s, "int") || !strings.Contains(s, "ok") {
			t.Fatalf("got %q", s)
		}
	})

	t.Run("Err_too_many_args", func(t *testing.T) {
		var v1 ast.ValueNode = ast.IntLiteralNode{Value: 1}
		var v2 ast.ValueNode = ast.IntLiteralNode{Value: 2}
		a1 := ast.ConstraintArgumentNode{Value: &v1}
		a2 := ast.ConstraintArgumentNode{Value: &v2}
		_, err := tr.transformResultIsDiscriminator(left, ast.ConstraintNode{Name: "Err", Args: []ast.ConstraintArgumentNode{a1, a2}})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("unknown_discriminator", func(t *testing.T) {
		_, err := tr.transformResultIsDiscriminator(left, ast.ConstraintNode{Name: "Nope"})
		if err == nil || !strings.Contains(err.Error(), "not Ok/Err") {
			t.Fatalf("unexpected: %v", err)
		}
	})

	t.Run("Ok_missing_value_arg", func(t *testing.T) {
		badArg := ast.ConstraintArgumentNode{}
		_, err := tr.transformResultIsDiscriminator(left, ast.ConstraintNode{Name: "Ok", Args: []ast.ConstraintArgumentNode{badArg}})
		if err == nil || !errors.Is(err, errMissingConstraintArgValue) {
			t.Fatalf("unexpected: %v", err)
		}
	})
}

func exprString(t *testing.T, e interface{}) string {
	t.Helper()
	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), e); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
