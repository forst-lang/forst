package forstpkg

import (
	"os"
	"path/filepath"
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

func TestPackageNameFromNodes_firstPackageWins(t *testing.T) {
	nodes := []ast.Node{
		ast.PackageNode{Ident: ast.Ident{ID: "demo"}},
		ast.PackageNode{Ident: ast.Ident{ID: "other"}},
		ast.FunctionNode{Ident: ast.Ident{ID: "F"}},
	}
	if got := PackageNameFromNodes(nodes); got != "demo" {
		t.Fatalf("expected first package clause name, got %q", got)
	}
}

func TestPackageNameFromNodes_noPackageDecl(t *testing.T) {
	nodes := []ast.Node{
		ast.FunctionNode{Ident: ast.Ident{ID: "main"}},
	}
	if got := PackageNameFromNodes(nodes); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestParseForstFile_roundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "a.ft")
	const content = `package main

func main() {
	println("x")
}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(nil)
	nodes, err := ParseForstFile(log, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) < 1 {
		t.Fatalf("expected nodes, got %d", len(nodes))
	}
	if PackageNameFromNodes(nodes) != "main" {
		t.Fatalf("package name: %#v", nodes)
	}
}

func TestParseAndMergePackage_twoFilesOnePackageClause(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	a := filepath.Join(dir, "a.ft")
	b := filepath.Join(dir, "b.ft")
	if err := os.WriteFile(a, []byte(`package main

func a(): Int { return 1 }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte(`package main

func b(): Int { return 2 }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(nil)
	merged, byPath, err := ParseAndMergePackage(log, []string{b, a})
	if err != nil {
		t.Fatal(err)
	}
	if len(byPath) != 2 {
		t.Fatalf("byPath: %d", len(byPath))
	}
	var pkgCount int
	for _, n := range merged {
		if _, ok := n.(ast.PackageNode); ok {
			pkgCount++
		}
	}
	if pkgCount != 1 {
		t.Fatalf("expected one package clause in merged AST, got %d", pkgCount)
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
