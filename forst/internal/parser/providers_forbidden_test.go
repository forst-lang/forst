package parser

import (
	"strings"
	"testing"
)

func TestParseProviders_forbiddenPostfixWith(t *testing.T) {
	t.Parallel()
	src := `package main
func TestX(t *testing.T) {
	needsLogger() with { Logger: &NopLogger {} }
}`
	err := parseShouldFail(src)
	if err == nil {
		t.Fatal("expected parse error for postfix with")
	}
	if !strings.Contains(err.Error(), "Parse error") {
		t.Fatalf("expected parse error, got: %v", err)
	}
}

func TestParseProviders_forbiddenAuthorUsesClause(t *testing.T) {
	t.Parallel()
	src := `package main
func expireToken(token Token) uses Logger, Clock {
	use logger: Logger
}`
	err := parseShouldFail(src)
	if err == nil {
		t.Fatal("expected parse error for author uses clause")
	}
}

func TestParseProviders_forbiddenWithForward(t *testing.T) {
	t.Parallel()
	src := `package main
func f() {
	with forward {
		x := 1
	}
}`
	err := parseShouldFail(src)
	if err == nil {
		t.Fatal("expected parse error for with forward")
	}
}

func TestParseProviders_forbiddenMapWiring(t *testing.T) {
	t.Parallel()
	src := `package main
func TestX(t *testing.T) {
	with map[String]Logger {
		x := 1
	}
}`
	err := parseShouldFail(src)
	if err == nil {
		t.Fatal("expected parse error for map wiring")
	}
}
