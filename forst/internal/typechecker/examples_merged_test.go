package typechecker

import (
	"path/filepath"
	"testing"
)

func TestCheckTypes_mergedProvidersDemo(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "examples", "in", "rfc", "providers")
	paths := []string{
		filepath.Join(root, "providers.ft"),
		filepath.Join(root, "providers_test.ft"),
		filepath.Join(root, "main_wiring.ft"),
	}
	tc := mergedPackageTypecheck(t, paths)
	if len(tc.FunctionProviders["expireToken"]) == 0 {
		t.Fatal("expireToken should declare provider slots")
	}
}

func TestCheckTypes_mergedImportsExample(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "examples", "in", "imports")
	paths := []string{
		filepath.Join(root, "main.ft"),
		filepath.Join(root, "greeting.ft"),
		filepath.Join(root, "cli.ft"),
		filepath.Join(root, "fmt_only.ft"),
	}
	tc := mergedPackageTypecheck(t, paths)
	if _, ok := tc.Functions["greeting"]; !ok {
		t.Fatal("greeting should be registered after merge")
	}
}

func TestCheckTypes_mergedTictactoePackage(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "examples", "in", "tictactoe")
	paths := []string{
		filepath.Join(root, "main", "engine.ft"),
		filepath.Join(root, "main", "server.ft"),
	}
	mergedPackageTypecheck(t, paths)
}
