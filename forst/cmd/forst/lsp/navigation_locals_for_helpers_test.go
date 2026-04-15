package lsp

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"

	"github.com/sirupsen/logrus"
)

func lexTokensForLSPHelperTest(source string) []ast.Token {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	return lexer.New([]byte(source), "test.ft", log).Lex()
}

func nthTokenIndexByType(tokens []ast.Token, tokenType ast.TokenIdent, n int) int {
	seen := 0
	for index := range tokens {
		if tokens[index].Type != tokenType {
			continue
		}
		if seen == n {
			return index
		}
		seen++
	}
	return -1
}

func TestNavigationHelpers_forHeaderScanners(t *testing.T) {
	tokens := lexTokensForLSPHelperTest(`package main

func f() {
  for idx, val := range arr {
    _ = idx
    _ = val
  }
  for i := 0; i < 2; i++ {
    _ = i
  }
}
`)

	rangeForIndex := nthTokenIndexByType(tokens, ast.TokenFor, 0)
	if rangeForIndex < 0 {
		t.Fatal("range for token not found")
	}
	rangeKey := findIdentBetweenForAndRange(tokens, rangeForIndex, "idx")
	if rangeKey == nil || rangeKey.Value != "idx" {
		t.Fatalf("expected range key idx token, got %+v", rangeKey)
	}
	rangeBodyL, rangeBodyR := forStmtBodyBraces(tokens, rangeForIndex)
	if rangeBodyL < 0 || rangeBodyR <= rangeBodyL {
		t.Fatalf("expected valid range for body braces, got l=%d r=%d", rangeBodyL, rangeBodyR)
	}

	threeClauseForIndex := nthTokenIndexByType(tokens, ast.TokenFor, 1)
	if threeClauseForIndex < 0 {
		t.Fatal("three-clause for token not found")
	}
	initIdentifier := findIdentInForThreeClauseHeader(tokens, threeClauseForIndex, "i")
	if initIdentifier == nil || initIdentifier.Value != "i" {
		t.Fatalf("expected for-init identifier i token, got %+v", initIdentifier)
	}
	lBrace := bodyLBraceAfterForHeader(tokens, threeClauseForIndex)
	if lBrace < 0 {
		t.Fatal("expected lbrace after for header")
	}
	threeBodyL, threeBodyR := forStmtBodyBraces(tokens, threeClauseForIndex)
	if threeBodyL < 0 || threeBodyR <= threeBodyL {
		t.Fatalf("expected valid three-clause for body braces, got l=%d r=%d", threeBodyL, threeBodyR)
	}
}

func TestNavigationHelpers_nthShortDeclIdentToken(t *testing.T) {
	tokens := lexTokensForLSPHelperTest(`package main

func f() {
  a := 1
  if true {
    a := 2
    _ = a
  }
}
`)

	first := nthShortDeclIdentToken(tokens, "a", 0)
	if first == nil || first.Value != "a" {
		t.Fatalf("expected first short decl token for a, got %+v", first)
	}
	second := nthShortDeclIdentToken(tokens, "a", 1)
	if second == nil || second.Value != "a" {
		t.Fatalf("expected second short decl token for a, got %+v", second)
	}
	third := nthShortDeclIdentToken(tokens, "a", 2)
	if third != nil {
		t.Fatalf("expected no third short decl token for a, got %+v", third)
	}
}
