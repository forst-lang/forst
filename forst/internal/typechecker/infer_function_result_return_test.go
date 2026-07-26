package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
)

func TestCheckTypes_resultReturn_mixedSuccessFailureArms(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	src := `package main

error ParseError {
	message: String,
}

func parse(input String): Result(String, ParseError) {
	if input == "" {
		return ParseError{ message: "empty" }
	}
	return input
}

func main() {
	println("ok")
}
`
	nodes, err := parser.NewTestParser(src, log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	sig := tc.Functions["parse"]
	if len(sig.ReturnTypes) != 1 || !sig.ReturnTypes[0].IsResultType() {
		t.Fatalf("parse return types = %v", sig.ReturnTypes)
	}
}
