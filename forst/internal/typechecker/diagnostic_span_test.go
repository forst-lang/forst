package typechecker

import (
	"errors"
	"strings"
	"testing"

	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestDiagnosticSpan_shapeUnknownFieldOnBadIdent(t *testing.T) {
	t.Parallel()
	src := `package main

type User = {
	name: String,
}

func main() {
	u := User { name: "a" }
	_ = u.nme
}
`
	diag := mustTypecheckDiagnostic(t, src)
	if diag.Code != "shape-unknown-field" {
		t.Fatalf("code = %q", diag.Code)
	}
	if !diag.Span.IsSet() {
		t.Fatal("expected set span on shape-unknown-field")
	}
	assertSpanNotOnPackage(t, src, diag)
	if !strings.Contains(srcLine(src, diag.Span.StartLine), "nme") {
		t.Fatalf("span line %d does not contain nme: %q span=%+v",
			diag.Span.StartLine, srcLine(src, diag.Span.StartLine), diag.Span)
	}
}

func TestDiagnosticSpan_resultOkOnNonResult(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	x := 1
	if x is Ok() {
	}
}
`
	diag := mustTypecheckDiagnostic(t, src)
	if diag.Code != "result-ok-subject" {
		t.Fatalf("code = %q want result-ok-subject; err=%s", diag.Code, diag.Error())
	}
	if !diag.Span.IsSet() {
		t.Fatal("expected set span on result-ok-subject")
	}
	assertSpanNotOnPackage(t, src, diag)
}

func mustTypecheckDiagnostic(t *testing.T, src string) *Diagnostic {
	t.Helper()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	nodes, err := parser.NewTestParser(src, log).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected typecheck error")
	}
	var diag *Diagnostic
	if !errors.As(err, &diag) || diag == nil {
		t.Fatalf("expected *Diagnostic, got %T: %v", err, err)
	}
	return diag
}

func assertSpanNotOnPackage(t *testing.T, src string, diag *Diagnostic) {
	t.Helper()
	if diag.Span.StartLine <= 1 {
		t.Fatalf("span on package line: %+v msg=%q", diag.Span, diag.Error())
	}
	line := strings.TrimSpace(srcLine(src, diag.Span.StartLine))
	if strings.HasPrefix(line, "package ") {
		t.Fatalf("span still on package statement: line=%q span=%+v", line, diag.Span)
	}
}

func srcLine(src string, line1 int) string {
	lines := strings.Split(src, "\n")
	if line1 < 1 || line1 > len(lines) {
		return ""
	}
	return lines[line1-1]
}
