package typechecker

import (
	"testing"

	"forst/internal/testutil"
)

func TestHarness_mustTypecheck_minimalProgram(t *testing.T) {
	src := `package main
func main() {}
`
	tc, nodes := MustTypecheck(t, src, testutil.TypecheckOpts{})
	if tc == nil || len(nodes) == 0 {
		t.Fatal("expected typechecker and nodes")
	}
}

func TestHarness_mustTypecheck_expectError(t *testing.T) {
	src := `package main
func main() {
	x := 1 + "two"
}
`
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{ExpectError: "type mismatch"})
	if tc == nil {
		t.Fatal("expected typechecker even on error path")
	}
}

func TestHarness_mustTypecheck_useModuleRoot(t *testing.T) {
	src := `package main
func main() {}
`
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
	if tc.GoWorkspaceDir == "" {
		t.Fatal("expected GoWorkspaceDir when UseModuleRoot is set")
	}
}

func TestHarness_mustTypecheckMixedPackage(t *testing.T) {
	root, importPath := testutil.WriteMixedGoForstModule(t, "mixed")
	src := `package memos

func main() {
	x := Add(1, 2)
	println(x)
}
`
	tc, nodes := MustTypecheckMixedPackage(t, root, importPath, src)
	if tc == nil || len(nodes) == 0 {
		t.Fatal("expected typechecker and nodes")
	}
}
