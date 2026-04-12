package hoverdoc

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestBuiltinTypeDocs_nonempty(t *testing.T) {
	t.Parallel()
	keys := []string{
		NameInt, NameFloat, NameString, NameBool, NameVoid,
		NameArray, NameMap, NameStruct,
		NameError, NamePointer, NameShape, NameResult, NameTuple, NameObject,
	}
	for _, k := range keys {
		if s := BuiltinTypeMarkdown(k); s == "" || !strings.Contains(s, k) {
			t.Fatalf("BuiltinTypeMarkdown(%q): empty or missing name: %q", k, s)
		}
	}
}

func TestMarkdownForKeywordToken_typesAndKeywords(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tok  ast.TokenIdent
		want string
	}{
		{ast.TokenInt, NameInt},
		{ast.TokenMap, NameMap},
		{ast.TokenEnsure, "ensure"},
		{ast.TokenIs, "is"},
	}
	for _, tc := range cases {
		got := MarkdownForKeywordToken(tc.tok)
		if got == "" || !strings.Contains(got, tc.want) {
			t.Fatalf("MarkdownForKeywordToken(%v): got %q, want substring %q", tc.tok, got, tc.want)
		}
	}
}

func TestGuardMarkdown_allBuiltinNames(t *testing.T) {
	t.Parallel()
	for _, name := range BuiltinGuardNames {
		if s := GuardMarkdown(name); s == "" || !strings.Contains(s, name) {
			t.Fatalf("GuardMarkdown(%q): %q", name, s)
		}
	}
}

func TestGuardMarkdown_unknown(t *testing.T) {
	t.Parallel()
	if GuardMarkdown("NotABuiltinGuard") != "" {
		t.Fatal("expected empty for unknown guard")
	}
}

func TestIsBuiltinTypeSurfaceName(t *testing.T) {
	t.Parallel()
	if !IsBuiltinTypeSurfaceName(NameResult) || IsBuiltinTypeSurfaceName("NotBuiltin") {
		t.Fatal("IsBuiltinTypeSurfaceName mismatch")
	}
}

func TestForstBlock(t *testing.T) {
	t.Parallel()
	s := forstBlock("a", "b")
	if !strings.Contains(s, "```forst") || !strings.Contains(s, "a\nb") {
		t.Fatalf("got %q", s)
	}
}

func TestMarkdownForKeywordToken_controlFlowAndLiterals(t *testing.T) {
	t.Parallel()
	cases := []ast.TokenIdent{
		ast.TokenIf, ast.TokenFor, ast.TokenNil, ast.TokenTrue, ast.TokenError,
	}
	for _, tok := range cases {
		if s := MarkdownForKeywordToken(tok); s == "" {
			t.Fatalf("empty for %v", tok)
		}
	}
}
