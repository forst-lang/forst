package lsp

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
)

func TestDiagnosticFromParseError_UsesLexerLineColumn(t *testing.T) {
	t.Parallel()
	pe := &parser.ParseError{
		Token: ast.Token{Line: 5, Column: 3, Value: "oops"},
		Msg:   "unexpected token",
	}
	d := DiagnosticFromParseError("file:///test.ft", pe)
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
