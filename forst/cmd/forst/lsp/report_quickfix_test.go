package lsp

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/diag"
	"forst/internal/typechecker"
)

func TestDiagnosticForTypecheckError_reportMessageAndFixes(t *testing.T) {
	t.Parallel()
	err := &typechecker.Diagnostic{
		Code:    "go-member-missing",
		Title:   "fmt.Printl not found",
		Problem: "No exported name `Printl` in package `fmt`.",
		Help:    "did you mean `Println`?",
		Span:    ast.SourceSpan{StartLine: 2, StartCol: 5, EndLine: 2, EndCol: 11},
		Fixes: []diag.Fix{{
			Title:   "Rename to Println",
			NewText: "Println",
			Span:    ast.SourceSpan{StartLine: 2, StartCol: 5, EndLine: 2, EndCol: 11},
		}},
	}
	d := diagnosticForTypecheckError("file:///t.ft", "package main\nfmt.Printl()\n", err, "forst-typechecker", "TYPE_MISMATCH")
	if d.Code != "go-member-missing" {
		t.Fatalf("code = %q", d.Code)
	}
	if !strings.Contains(d.Message, "error[go-member-missing]:") {
		t.Fatalf("message = %q", d.Message)
	}
	if !strings.Contains(d.Message, "help: did you mean") {
		t.Fatalf("message missing help: %q", d.Message)
	}
	if d.Data == nil {
		t.Fatal("expected data.fixes")
	}
	acts := reportFixQuickFixActions("file:///t.ft", []codeActionDiagnosticParam{{
		Range:   d.Range,
		Message: d.Message,
		Code:    d.Code,
		Data:    d.Data,
	}})
	if len(acts) != 1 {
		t.Fatalf("actions = %d", len(acts))
	}
	if acts[0].Title != "Rename to Println" {
		t.Fatalf("title = %q", acts[0].Title)
	}
	edits := acts[0].Edit.Changes["file:///t.ft"]
	if len(edits) != 1 || edits[0].NewText != "Println" {
		t.Fatalf("edits = %+v", edits)
	}
}
