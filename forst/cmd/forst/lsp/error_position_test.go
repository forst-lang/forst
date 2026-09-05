package lsp

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/diag"
	"forst/internal/parser"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func TestDiagnosticFromParseError_UsesLexerLineColumn(t *testing.T) {
	t.Parallel()
	pe := &parser.ParseError{
		Token: ast.Token{Line: 5, Column: 3, Value: "oops"},
		Msg:   "unexpected token",
	}
	d := DiagnosticFromParseError("file:///test.ft", pe)
	if d.Code != ErrorCodeUnexpectedToken {
		t.Fatalf("code = %q want %q", d.Code, ErrorCodeUnexpectedToken)
	}
	if d.Range.Start.Line != 4 {
		t.Errorf("LSP line = %d, want 4 (lexer line 5 → 0-based 4)", d.Range.Start.Line)
	}
	if d.Range.Start.Character != 2 {
		t.Errorf("start character = %d, want 2 (column 3 → 0-based 2)", d.Range.Start.Character)
	}
	wantEnd := 2 + len([]rune("oops"))
	if d.Range.End.Character != wantEnd {
		t.Errorf("end character = %d, want %d", d.Range.End.Character, wantEnd)
	}
}

func TestBestEffortLineColumnFromErrorMessage_FunctionName(t *testing.T) {
	t.Parallel()
	content := "package main\n\nfunc Broken() {\n  return 1\n}\n"
	errMsg := "inferred return type ... of function Broken does not match parsed return type ..."
	line, col := bestEffortLineColumnFromErrorMessage(content, errMsg)
	if line != 3 || col != 1 {
		t.Fatalf("got line %d col %d, want line 3 col 1", line, col)
	}
}

func TestBestEffortLineColumnFromErrorMessage_UndefinedSymbol(t *testing.T) {
	t.Parallel()
	content := "package main\n\nfunc main() {\n    ensure err asdf\n}\n"
	errMsg := "undefined symbol: asdf [scope: Scope(EnsureBlock)]"
	line, col := bestEffortLineColumnFromErrorMessage(content, errMsg)
	if line != 4 {
		t.Fatalf("line = %d, want 4", line)
	}
	wantCol := strings.Index(strings.Split(content, "\n")[3], "asdf") + 1
	if col != wantCol {
		t.Fatalf("col = %d, want %d", col, wantCol)
	}
}

func TestParseErrorLocationFromMessage(t *testing.T) {
	t.Parallel()
	msg := "\nParse error in x.ft:7:12 (at line 7, column 12):\nbad"
	l, c, ok := parseErrorLocationFromMessage(msg)
	if !ok || l != 7 || c != 12 {
		t.Fatalf("got %d,%d ok=%v", l, c, ok)
	}
}

func TestParseErrorLocationFromMessage_ParseErrorAtForm(t *testing.T) {
	t.Parallel()
	// Matches parser.ParseError.Error() via fmt.Errorf("parser panic: %%v", pe)
	msg := "parser panic: Parse error at f1:26:16: Expected token type 'IDENTIFIER' but got 'DOT'"
	l, c, ok := parseErrorLocationFromMessage(msg)
	if !ok || l != 26 || c != 16 {
		t.Fatalf("got line %d col %d ok=%v", l, c, ok)
	}
}

func TestDiagnosticForTypecheckError_usesStructuredSpan(t *testing.T) {
	t.Parallel()
	err := &typechecker.Diagnostic{
		Title: "wrong type",
		Help:  "fix the Go call",
		Code:  "go-call",
		Span:  ast.SourceSpan{StartLine: 2, StartCol: 5, EndLine: 2, EndCol: 9},
	}
	d := diagnosticForTypecheckError("file:///t.ft", "", err, "forst-typechecker", ErrorCodeTypeMismatch)
	if d.Range.Start.Line != 1 || d.Range.Start.Character != 4 {
		t.Fatalf("start = %+v", d.Range.Start)
	}
	if d.Range.End.Line != 1 || d.Range.End.Character != 8 {
		t.Fatalf("end = %+v", d.Range.End)
	}
	if d.Code != "go-call" {
		t.Fatalf("code = %q", d.Code)
	}
}

func TestDiagnosticForTypecheckError_providersObligationRelated(t *testing.T) {
	t.Parallel()
	err := &typechecker.Diagnostic{
		Title: "needsLogger requires Logger; not supplied",
		Help:  "add Logger to the with { … } block or ciProviders bundle",
		Code:  "providers-unsatisfied",
		Span:  ast.SourceSpan{StartLine: 10, StartCol: 2, EndLine: 10, EndCol: 15},
		Related: []typechecker.RelatedDiagnostic{
			{Msg: "caller TestX", Span: ast.SourceSpan{StartLine: 8, StartCol: 1, EndLine: 8, EndCol: 5}},
			{Msg: "needsLogger declares Providers requirements", Span: ast.SourceSpan{StartLine: 4, StartCol: 1, EndLine: 4, EndCol: 12}},
		},
	}
	d := diagnosticForTypecheckError("file:///t.ft", "", err, "forst-typechecker", ErrorCodeTypeMismatch)
	if len(d.RelatedInformation) != 2 {
		t.Fatalf("related = %d", len(d.RelatedInformation))
	}
	if d.RelatedInformation[0].Location.Range.Start.Line != 7 {
		t.Fatalf("caller link line = %d", d.RelatedInformation[0].Location.Range.Start.Line)
	}
	if d.RelatedInformation[1].Message != "needsLogger declares Providers requirements" {
		t.Fatalf("callee message = %q", d.RelatedInformation[1].Message)
	}
}

func TestDiagnosticForTypecheckOrTransform_plainError(t *testing.T) {
	t.Parallel()
	content := "package main\n\nfunc main() {\n  x\n}\n"
	err := errors.New("transform failed: something")
	d := diagnosticForTypecheckOrTransform("file:///t.ft", content, err, "forst-transformer", ErrorCodeTransformationFailed)
	if d.Source != "forst-transformer" || d.Code != ErrorCodeTransformationFailed {
		t.Fatalf("source=%q code=%q", d.Source, d.Code)
	}
	if !strings.Contains(d.Message, "transform failed") {
		t.Fatalf("message = %q", d.Message)
	}
}

func TestDiagnosticForTypecheckError_plainErrorFallsBackToBestEffort(t *testing.T) {
	t.Parallel()
	content := "package main\n\nfunc main() {\n  fooBar\n}\n"
	err := errors.New("undefined symbol: fooBar [scope: main]")
	d := diagnosticForTypecheckError("file:///t.ft", content, err, "forst-typechecker", ErrorCodeTypeMismatch)
	if d.Range.Start.Line != 3 {
		t.Fatalf("start line = %d", d.Range.Start.Line)
	}
}

func TestAtoiLineCol_invalidLine(t *testing.T) {
	t.Parallel()
	if _, _, ok := atoiLineCol("0", "1"); ok {
		t.Fatal("expected false for line 0")
	}
}

func TestDiagnosticForParseFailure_ParseError(t *testing.T) {
	t.Parallel()
	pe := &parser.ParseError{
		Token: ast.Token{Line: 2, Column: 4, Value: "oops"},
		Msg:   "bad token",
	}
	d := diagnosticForParseFailure("file:///t.ft", "package main\n\nx\n", pe)
	if d.Source != "forst-parser" || d.Code != ErrorCodeInvalidSyntax {
		t.Fatalf("got source=%q code=%q", d.Source, d.Code)
	}
	if d.Range.Start.Line != 1 || d.Range.Start.Character != 3 {
		t.Fatalf("range start = %+v", d.Range.Start)
	}
}

func TestLspCodeForParseMessage_wrappedExpectedVsFound(t *testing.T) {
	t.Parallel()
	msg := "Parse error at f.ft:1:2: expected IDENTIFIER, found LBRACE (current text \"{\")"
	if c := lspCodeForParseMessage(msg); c != ErrorCodeUnexpectedToken {
		t.Fatalf("code = %q want %q", c, ErrorCodeUnexpectedToken)
	}
}

func TestDiagnosticFromParseError_expectedVsFoundMessageCode(t *testing.T) {
	t.Parallel()
	pe := &parser.ParseError{
		Token: ast.Token{Line: 1, Column: 1, Value: "{"},
		Msg:   `expected IDENTIFIER, found LBRACE (current text "{")`,
	}
	d := DiagnosticFromParseError("file:///t.ft", pe)
	if d.Code != ErrorCodeUnexpectedToken {
		t.Fatalf("code = %q", d.Code)
	}
	if !strings.Contains(d.Message, "expected ") || !strings.Contains(d.Message, "found ") {
		t.Fatalf("message = %q", d.Message)
	}
}

func TestDiagnosticForParseFailure_legacyFileLineColInMessage(t *testing.T) {
	t.Parallel()
	err := errors.New("Parse error in t.ft:2:3: something went wrong")
	d := diagnosticForParseFailure("file:///t.ft", "package main\nxx\n", err)
	if d.Range.Start.Line != 1 || d.Range.Start.Character != 2 {
		t.Fatalf("want line 1 col 2, got %+v", d.Range.Start)
	}
}

func TestBestEffortLineColumnFromErrorMessage_emptyContent(t *testing.T) {
	t.Parallel()
	line, col := bestEffortLineColumnFromErrorMessage("", "undefined symbol: x")
	if line != 1 || col != 1 {
		t.Fatalf("got %d,%d", line, col)
	}
}

func TestBestEffortLineColumnFromErrorMessage_typeField(t *testing.T) {
	t.Parallel()
	content := "package main\n\ntype Box = {\n  n: Int,\n}\n"
	errMsg := `type Box field "n" invalid`
	line, col := bestEffortLineColumnFromErrorMessage(content, errMsg)
	if line != 3 || col != 1 {
		t.Fatalf("got line %d col %d", line, col)
	}
}

func TestBestEffortLineColumnFromErrorMessage_functionParameter(t *testing.T) {
	t.Parallel()
	content := "package main\n\nfunc F(x Int) {}\n"
	errMsg := "function F parameter mismatch"
	line, col := bestEffortLineColumnFromErrorMessage(content, errMsg)
	if line != 3 || col != 1 {
		t.Fatalf("got line %d col %d", line, col)
	}
}

func TestBestEffortLineColumnFromErrorMessage_functionReturn(t *testing.T) {
	t.Parallel()
	content := "package main\n\nfunc G(): Int {\n  return 1\n}\n"
	errMsg := "function G return type wrong"
	line, col := bestEffortLineColumnFromErrorMessage(content, errMsg)
	if line != 3 || col != 1 {
		t.Fatalf("got line %d col %d", line, col)
	}
}

func TestBestEffortLineColumnFromErrorMessage_undefinedFunction(t *testing.T) {
	t.Parallel()
	content := "package main\n\nfunc main() {\n  missing()\n}\n"
	line, col := bestEffortLineColumnFromErrorMessage(content, "undefined function: missing")
	if line != 4 {
		t.Fatalf("line = %d", line)
	}
	if want := strings.Index(strings.Split(content, "\n")[3], "missing") + 1; col != want {
		t.Fatalf("col = %d want %d", col, want)
	}
}

func TestBestEffortLineColumnFromErrorMessage_unknownIdentifier(t *testing.T) {
	t.Parallel()
	content := "package main\n\nfunc main() {\n  z := 1\n}\n"
	line, col := bestEffortLineColumnFromErrorMessage(content, "unknown identifier: z")
	if line != 4 || col != 3 {
		t.Fatalf("got line %d col %d", line, col)
	}
}

func TestBestEffortLineColumnFromErrorMessage_undeclaredVariableQuote(t *testing.T) {
	t.Parallel()
	content := "package main\n\nfunc main() {\n  foo := 1\n}\n"
	line, col := bestEffortLineColumnFromErrorMessage(content, "undeclared variable 'foo'")
	if line != 4 {
		t.Fatalf("line = %d", line)
	}
	if want := strings.Index(strings.Split(content, "\n")[3], "foo") + 1; col != want {
		t.Fatalf("col = %d want %d", col, want)
	}
}

func TestBestEffortLineColumnFromErrorMessage_shapeUnknownFieldReport(t *testing.T) {
	t.Parallel()
	content := "package main\n\ntype User = {\n  name: String,\n}\n\nfunc main() {\n  u := User { name: \"a\" }\n  _ = u.nme\n}\n"
	errMsg := "error[shape-unknown-field]: no field named \"nme\"\n\n  Type `User` has no field `nme`."
	line, col := bestEffortLineColumnFromErrorMessage(content, errMsg)
	if line != 9 {
		t.Fatalf("line = %d want 9", line)
	}
	if want := strings.Index(strings.Split(content, "\n")[8], "nme") + 1; col != want {
		t.Fatalf("col = %d want %d", col, want)
	}
}

func TestBestEffortLineColumnFromErrorMessage_goExportedNameReport(t *testing.T) {
	t.Parallel()
	content := "package main\n\nimport \"fmt\"\n\nfunc main() {\n  fmt.Printl(\"hi\")\n}\n"
	errMsg := "error[go-member-missing]: fmt.Printl not found\n\n  No exported name `Printl` in package `fmt`."
	line, col := bestEffortLineColumnFromErrorMessage(content, errMsg)
	if line != 6 {
		t.Fatalf("line = %d want 6", line)
	}
	if want := strings.Index(strings.Split(content, "\n")[5], "Printl") + 1; col != want {
		t.Fatalf("col = %d want %d", col, want)
	}
}

func lspFirstNonPackageLine(content string) int {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "package ") {
			continue
		}
		return i
	}
	return 0
}

func assertDiagnosticNotOnPackageLine(t *testing.T, content string, d LSPDiagnostic) {
	t.Helper()
	minLine := lspFirstNonPackageLine(content)
	if d.Range.Start.Line < minLine {
		t.Fatalf("diagnostic on package line %d (first non-package line %d): start=%+v msg=%q",
			d.Range.Start.Line, minLine, d.Range.Start, d.Message)
	}
}

func TestDiagnosticForTypecheckError_logsWhenSpanMissing(t *testing.T) {
	t.Parallel()
	prev := errorPositionLog
	defer func() { errorPositionLog = prev }()
	buf := &bytes.Buffer{}
	log := logrus.New()
	log.SetOutput(buf)
	log.SetLevel(logrus.ErrorLevel)
	errorPositionLog = log

	err := &typechecker.Diagnostic{
		Code:    "test-missing-span",
		Title:   "no span",
		Problem: "undefined symbol: fooBar [scope: main]",
		Help:    "fix it",
	}
	content := "package main\n\nfunc main() {\n  fooBar\n}\n"
	d := diagnosticForTypecheckError("file:///t.ft", content, err, "forst-typechecker", ErrorCodeTypeMismatch)
	if d.Range.Start.Line != 3 {
		t.Fatalf("expected heuristic line 3 (fooBar), got %+v", d.Range.Start)
	}
	if !strings.Contains(buf.String(), "without span") {
		t.Fatalf("expected error log, got %q", buf.String())
	}
}

func TestDiagnosticForTypecheckError_isOkNotOnPackageLine(t *testing.T) {
	t.Parallel()
	s, dir := NewTestLSPServer(t, "lsp_span_guard")
	src := `package main

func main() {
	x := 1
	if x is Ok() {
	}
}
`
	uri := OpenTestBuffer(t, s, dir, "main.ft", src)
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.CheckErr == nil {
		t.Fatalf("expected typecheck error, ctx=%v err=%v", ctx, ctx.CheckErr)
	}
	d := diagnosticForTypecheckError(uri, src, ctx.CheckErr, "forst-typechecker", ErrorCodeTypeMismatch)
	assertDiagnosticNotOnPackageLine(t, src, d)
	if !strings.Contains(d.Message, "result-ok-subject") {
		t.Fatalf("message = %q", d.Message)
	}
}

func TestDiagnosticForTypecheckError_shapeUnknownFieldNotOnPackageLine(t *testing.T) {
	t.Parallel()
	s, dir := NewTestLSPServer(t, "lsp_span_guard_shape")
	src := `package main

type User = {
	name: String,
}

func main() {
	u := User { name: "a" }
	_ = u.nme
}
`
	uri := OpenTestBuffer(t, s, dir, "main.ft", src)
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.CheckErr == nil {
		t.Fatalf("expected typecheck error, ctx=%v err=%v", ctx, ctx.CheckErr)
	}
	d := diagnosticForTypecheckError(uri, src, ctx.CheckErr, "forst-typechecker", ErrorCodeTypeMismatch)
	assertDiagnosticNotOnPackageLine(t, src, d)
	if d.Code != "shape-unknown-field" {
		t.Fatalf("code = %q", d.Code)
	}
	var td *typechecker.Diagnostic
	if errors.As(ctx.CheckErr, &td) && td != nil && !td.Span.IsSet() {
		t.Fatal("typechecker Diagnostic span must be set for shape-unknown-field")
	}
}

func TestDiagnosticForTypecheckError_goMemberMissingNotOnPackageLine(t *testing.T) {
	t.Parallel()
	s, dir := NewTestLSPServer(t, "lsp_span_guard_go")
	src := `package main

import "fmt"

func main() {
	fmt.Printl("hi")
}
`
	uri := OpenTestBuffer(t, s, dir, "main.ft", src)
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.CheckErr == nil {
		t.Fatalf("expected typecheck error, ctx=%v err=%v", ctx, ctx.CheckErr)
	}
	d := diagnosticForTypecheckError(uri, src, ctx.CheckErr, "forst-typechecker", ErrorCodeTypeMismatch)
	assertDiagnosticNotOnPackageLine(t, src, d)
	if d.Code != "go-member-missing" {
		t.Fatalf("code = %q", d.Code)
	}
}

func TestDiagnosticFromParseError_failWithReportMessageCode(t *testing.T) {
	t.Parallel()
	msg := diag.FormatReport(diag.Report{
		Code:    "typeguard-forbidden-use",
		Title:   "use statement not allowed in type guards",
		Problem: "use is not valid inside a type guard body.",
		Help:    "move the use to the enclosing function.",
	})
	pe := &parser.ParseError{
		Token: ast.Token{Line: 4, Column: 2, Value: "use"},
		Msg:   msg,
	}
	d := DiagnosticFromParseError("file:///t.ft", pe)
	if d.Code != "typeguard-forbidden-use" {
		t.Fatalf("code = %q", d.Code)
	}
	if d.Range.Start.Line != 3 {
		t.Fatalf("start line = %d want 3", d.Range.Start.Line)
	}
	if !strings.Contains(d.Message, "error[typeguard-forbidden-use]:") {
		t.Fatalf("message = %q", d.Message)
	}
}
