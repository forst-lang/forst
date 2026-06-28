package typechecker

import (
	"testing"
)

func TestProviders_cycle_intraPackagePreservesLogger(t *testing.T) {
	src := `package main

type Logger = { info(msg String) }

func f() {
	g()
}

func g() {
	use logger: Logger
	f()
}
`
	tc, err := parseAndCheck(t, src)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	if !containsAll(providerRootNames(tc.FunctionProviders["f"]), "Logger") {
		t.Fatalf("f should require Logger, got %v", tc.FunctionProviders["f"])
	}
	if !containsAll(providerRootNames(tc.FunctionProviders["g"]), "Logger") {
		t.Fatalf("g should require Logger, got %v", tc.FunctionProviders["g"])
	}
}
