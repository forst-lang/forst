package parser

import (
	"fmt"
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestParseFile_errorNominal_missingClosingBrace_reportsLocation(t *testing.T) {
	t.Parallel()
	src := `package main

error ParseError {
	message: String,

func main() {}
`
	err := parseShouldFail(src)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "Parse error at") {
		t.Fatalf("expected location in error, got: %v", err)
	}
}

func TestParseFile_ensureOrMalformedPayload_reportsUsefulError(t *testing.T) {
	t.Parallel()
	src := `package main

func run(x Int): Result(Int, Error) {
	ensure x is GreaterThan(1) or
	return x
}
`
	err := parseShouldFail(src)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "expected") {
		t.Fatalf("expected parser expectation error, got: %v", err)
	}
}

func TestParseFile_elseIfMissingBlock_reportsTokenLocation(t *testing.T) {
	t.Parallel()
	src := `package main

func run(n Int) {
	if n > 0 {
		println("ok")
	} else if n < 0
		println("bad")
}
`
	err := parseShouldFail(src)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), ":7:3:") {
		t.Fatalf("expected column in parser error, got: %v", err)
	}
}

func parseShouldFail(src string) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%v", recovered)
		}
	}()
	_, err = NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	return err
}
