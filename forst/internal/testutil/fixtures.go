package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// FixtureProvidersCrossPkg returns the canonical cross_pkg providers example root.
func FixtureProvidersCrossPkg(tb testing.TB) string {
	tb.Helper()
	root := filepath.Join("..", "..", "..", "examples", "in", "rfc", "providers", "cross_pkg")
	if _, err := os.Stat(root); err != nil {
		tb.Fatalf("cross_pkg fixture: %v", err)
	}
	return root
}

// FixtureSharedGoImports returns a temp module root suitable for fmt/os/exec import tests.
func FixtureSharedGoImports(tb testing.TB) string {
	tb.Helper()
	return WriteModule(tb, "shared_go_imports", map[string]string{
		"probe.ft": `package main

import "fmt"

func probe(): String {
	return fmt.Sprint("ok")
}
`,
	})
}
