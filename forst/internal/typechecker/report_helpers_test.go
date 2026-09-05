package typechecker

import (
	"errors"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/diag"
)

func TestShapeUnknownFieldError_reportAndFix(t *testing.T) {
	t.Parallel()
	err := shapeUnknownFieldError(ast.TypeIdent("User"), "nme", []string{"name", "age", "email"}, ast.SourceSpan{
		StartLine: 1, StartCol: 2, EndLine: 1, EndCol: 5,
	})
	msg := err.Error()
	if !strings.Contains(msg, "shape-unknown-field") {
		t.Fatalf("code missing: %s", msg)
	}
	if !strings.Contains(msg, "did you mean `name`?") {
		t.Fatalf("help missing: %s", msg)
	}
	if !strings.Contains(msg, "known fields:") {
		t.Fatalf("note missing: %s", msg)
	}
	var d *Diagnostic
	if !errors.As(err, &d) || d == nil {
		t.Fatal("expected *Diagnostic")
	}
	if len(d.Fixes) != 1 || d.Fixes[0].NewText != "name" {
		t.Fatalf("fixes = %+v", d.Fixes)
	}
}

func TestGoMemberMissingError_reportAndFix(t *testing.T) {
	t.Parallel()
	err := goMemberMissingError("fmt", "Printl", []string{"Print", "Printf", "Println"}, ast.SourceSpan{
		StartLine: 1, StartCol: 5, EndLine: 1, EndCol: 11,
	})
	msg := err.Error()
	if !strings.Contains(msg, "go-member-missing") {
		t.Fatalf("code missing: %s", msg)
	}
	if !strings.Contains(msg, "did you mean `Println`?") {
		t.Fatalf("help missing: %s", msg)
	}
	var d *Diagnostic
	if !errors.As(err, &d) || d == nil {
		t.Fatal("expected *Diagnostic")
	}
	if len(d.Fixes) != 1 || d.Fixes[0].NewText != "Println" {
		t.Fatalf("fixes = %+v", d.Fixes)
	}
	_ = diag.FormatReport(d.Report())
}

func TestShapeUnknownFieldError_builtinTypeUsesSurfaceName(t *testing.T) {
	t.Parallel()
	err := shapeUnknownFieldError(ast.TypeInt, "test", nil, ast.SourceSpan{
		StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 5,
	})
	msg := err.Error()
	if strings.Contains(msg, "TYPE_INT") {
		t.Fatalf("leaked internal type ident: %s", msg)
	}
	if !strings.Contains(msg, "Type `Int` has no field `test`") {
		t.Fatalf("expected Int surface name: %s", msg)
	}
}

func TestFormatTypeIdentForDiag_builtins(t *testing.T) {
	t.Parallel()
	if got := formatTypeIdentForDiag(ast.TypeInt); got != "Int" {
		t.Fatalf("got %q", got)
	}
	if got := formatTypeIdentForDiag(ast.TypeString); got != "String" {
		t.Fatalf("got %q", got)
	}
}
