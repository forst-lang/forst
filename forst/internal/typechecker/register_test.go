package typechecker

import (
	"testing"

	"forst/internal/parser"
)

// TestRegister_typeAliasStoredInDefs verifies registerType from the collect phase stores definitions.
func TestRegister_typeAliasStoredInDefs(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

type Phone = String

func main() {
	println("ok")
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
	def, ok := tc.Defs["Phone"]
	if !ok {
		t.Fatal("expected Phone in Defs")
	}
	if def == nil {
		t.Fatal("Phone def is nil")
	}
}
