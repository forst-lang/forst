package forstpkg

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestMergePackageASTs_skipsExtraPackageNodes(t *testing.T) {
	a := []ast.Node{ast.PackageNode{Ident: ast.Ident{ID: "demo"}}}
	b := []ast.Node{
		ast.PackageNode{Ident: ast.Ident{ID: "demo"}},
		ast.FunctionNode{Ident: ast.Ident{ID: "F"}},
	}
	merged := MergePackageASTs([][]ast.Node{a, b})
	var pkgCount int
	for _, n := range merged {
		if _, ok := n.(ast.PackageNode); ok {
			pkgCount++
		}
	}
	if pkgCount != 1 {
		t.Fatalf("expected one PackageNode, got %d", pkgCount)
	}
	if len(merged) != 2 {
		t.Fatalf("expected package + one function node, got %d nodes", len(merged))
	}
}

func TestPackageNameOrDefault(t *testing.T) {
	if PackageNameOrDefault("") != "main" {
		t.Fatal()
	}
	if PackageNameOrDefault("x") != "x" {
		t.Fatal()
	}
}

func TestParseAndMergePackage_emptyPaths(t *testing.T) {
	log := logrus.New()
	log.SetOutput(nil)
	merged, byPath, err := ParseAndMergePackage(log, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 0 || len(byPath) != 0 {
		t.Fatalf("expected empty, got merged=%d byPath=%d", len(merged), len(byPath))
	}
}
