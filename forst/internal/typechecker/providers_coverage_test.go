package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
)

func TestProvidersEngine_collectUseSitesInFunction(t *testing.T) {
	src := `package main

import "testing"

type Logger = { info(msg String) }
type NopLogger = {}

func (NopLogger) info(msg String) {}

func work() {
	use logger: Logger
	logger.info("x")
}

func TestX(t *testing.T) {
	with { Logger: &NopLogger {} } {
		work()
	}
}
`
	tc, err := parseAndCheck(t, src)
	if err != nil {
		t.Fatal(err)
	}
	slots := tc.FunctionProviders["work"]
	if len(slots) != 1 || slots[0].RootIdent != "Logger" {
		t.Fatalf("work providers = %#v", slots)
	}
}

func TestProvidersScope_mergeNestedWithSubtractsSatisfied(t *testing.T) {
	src := `package main

import "testing"

type Logger = { info(msg String) }
type NopLogger = { info(msg String) }

func inner() {
	use logger: Logger
}

func TestX(t *testing.T) {
	with { Logger: NopLogger{} } {
		inner()
	}
}
`
	tc, err := parseAndCheck(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if len(tc.FunctionProviders["TestX"]) != 0 {
		t.Fatalf("TestX should not inherit Logger when inner scope satisfies: %#v", tc.FunctionProviders["TestX"])
	}
}

func TestLookupFunction_resolveCallInSamePackage(t *testing.T) {
	log := setupTestLogger(nil)
	src := `package main

func helper(): Int {
	return 1
}

func main() {
	x := helper()
	println(x)
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	if _, ok := tc.Functions[ast.Identifier("helper")]; !ok {
		t.Fatal("helper should be registered")
	}
}
