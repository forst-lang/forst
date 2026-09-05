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
	err := shapeUnknownFieldError("User", "nme", []string{"name", "age", "email"}, ast.SourceSpan{
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

func TestGuardUndefinedError(t *testing.T) {
	t.Parallel()
	err := guardUndefinedError("Adult", ast.SourceSpan{
		StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 6,
	})
	if !strings.Contains(err.Error(), "guard-undefined") {
		t.Fatalf("%v", err)
	}
	if strings.Contains(err.Error(), "%T") || strings.Contains(err.Error(), "defs has") {
		t.Fatalf("leaked internals: %v", err)
	}
}
