package testutil

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestModuleRoot_findsGoMod(t *testing.T) {
	root := ModuleRoot(t)
	if !strings.HasSuffix(root, "forst") && !strings.Contains(root, "forst") {
		t.Fatalf("unexpected module root: %s", root)
	}
}

func TestAssertSingleType(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		AssertSingleTypeString(t, []ast.TypeNode{{Ident: "Int"}}, "Int")
	})
}

func TestWriteMixedGoForstModule(t *testing.T) {
	root, importPath := WriteMixedGoForstModule(t, "mixed")
	if root == "" || importPath != "mixedtest/mixed" {
		t.Fatalf("got root=%q importPath=%q", root, importPath)
	}
}

func TestParseSource(t *testing.T) {
	src := `package main
func main() {}
`
	nodes := ParseSource(t, src, "test.ft", nil)
	if len(nodes) == 0 {
		t.Fatal("expected nodes")
	}
}

func TestExamplePath(t *testing.T) {
	path := ExamplePath(t, "basic.ft")
	if !strings.HasSuffix(path, "basic.ft") {
		t.Fatalf("unexpected path: %s", path)
	}
}
