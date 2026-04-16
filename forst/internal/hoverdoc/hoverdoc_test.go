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

func TestGuardMarkdownQualified_prefixesBaseType(t *testing.T) {
	t.Parallel()
	md := GuardMarkdownQualified("Int", GuardGreaterThan)
	if !strings.HasPrefix(md, "**`Int.GreaterThan`** (guard)") {
		t.Fatalf("want Int.GreaterThan title, got:\n%s", md)
	}
	if strings.Contains(md, "**`GreaterThan`** (guard)") {
		t.Fatal("should not use unqualified title when base is set")
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

func TestForstBlock_emptyVariadic(t *testing.T) {
	t.Parallel()
	if forstBlock() != "" {
		t.Fatal("expected empty for no lines")
	}
}

func TestGuardMarkdownQualified_unknown_returnsEmpty(t *testing.T) {
	t.Parallel()
	if GuardMarkdownQualified("Int", "NotARealGuard") != "" {
		t.Fatal("expected empty when guard name unknown")
	}
}

func TestGuardMarkdownQualified_nonMatchingTitlePrefix_returnsDocUnchanged(t *testing.T) {
	orig := guardDocs[GuardMin]
	guardDocs[GuardMin] = "no standard title prefix\n\nbody"
	defer func() { guardDocs[GuardMin] = orig }()

	md := GuardMarkdownQualified("Int", GuardMin)
	if md != guardDocs[GuardMin] {
		t.Fatalf("expected doc unchanged, got %q", md)
	}
}

func TestGuardMarkdownQualified_emptyBase_matchesUnqualifiedPrefix(t *testing.T) {
	t.Parallel()
	md := GuardMarkdownQualified("   ", GuardMin)
	if !strings.HasPrefix(md, "**`Min`** (guard)") {
		t.Fatalf("expected unqualified title, got:\n%s", md)
	}
}

func TestBuiltinTypeMarkdown_unknownReturnsEmpty(t *testing.T) {
	t.Parallel()
	if BuiltinTypeMarkdown("TotallyUnknownType") != "" {
		t.Fatal("expected empty")
	}
}

func TestMarkdownForKeywordToken_switch(t *testing.T) {
	t.Parallel()
	// One token from each major branch of MarkdownForKeywordToken (builtins/types vs keywords).
	cases := []ast.TokenIdent{
		ast.TokenFloat, ast.TokenString, ast.TokenBool, ast.TokenVoid,
		ast.TokenArray, ast.TokenMap,
		ast.TokenStruct, ast.TokenFunc, ast.TokenType, ast.TokenReturn,
		ast.TokenImport, ast.TokenPackage, ast.TokenEnsure, ast.TokenIs,
		ast.TokenOr, ast.TokenElseIf, ast.TokenElse, ast.TokenRange,
		ast.TokenBreak, ast.TokenContinue, ast.TokenSwitch, ast.TokenCase,
		ast.TokenDefault, ast.TokenFallthrough, ast.TokenVar, ast.TokenConst,
		ast.TokenChan, ast.TokenInterface, ast.TokenGo, ast.TokenDefer,
		ast.TokenGoto, ast.TokenArrow, ast.TokenLogicalAnd, ast.TokenLogicalOr,
		ast.TokenLogicalNot,
	}
	for _, tok := range cases {
		if s := MarkdownForKeywordToken(tok); s == "" {
			t.Fatalf("empty markdown for token %v", tok)
		}
	}
}

func TestMarkdownForKeywordToken_controlFlowAndLiterals(t *testing.T) {
	t.Parallel()
	cases := []ast.TokenIdent{
		ast.TokenIf, ast.TokenFor, ast.TokenNil, ast.TokenTrue, ast.TokenFalse, ast.TokenError,
	}
	for _, tok := range cases {
		if s := MarkdownForKeywordToken(tok); s == "" {
			t.Fatalf("empty for %v", tok)
		}
	}
}

func TestMarkdownForKeywordToken_unknownReturnsEmpty(t *testing.T) {
	t.Parallel()
	if MarkdownForKeywordToken(ast.TokenIdent("__not_a_keyword__")) != "" {
		t.Fatal("expected empty markdown for unknown token ident")
	}
}
