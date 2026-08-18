package semantic_test

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/semantic"
)

func TestRouterFixture_parsesCatalogAsAssertionWithRouter(t *testing.T) {
	path := mustAbs(t, "testdata/router/catalog/api.ft")
	nodes, err := forstpkg.ParseForstFile(nil, path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, n := range nodes {
		td, ok := n.(ast.TypeDefNode)
		if !ok || td.Ident != "Catalog" {
			continue
		}
		ae, ok := td.Expr.(ast.TypeDefAssertionExpr)
		if !ok {
			t.Fatalf("Catalog expr = %T, want TypeDefAssertionExpr", td.Expr)
		}
		var hasRouter bool
		for _, c := range ae.Assertion.Constraints {
			if c.Name == "Router" {
				hasRouter = true
			}
		}
		if !hasRouter {
			t.Fatalf("Router missing from constraints: %#v", ae.Assertion.Constraints)
		}
		return
	}
	t.Fatal("Catalog typedef not found")
}

func TestRouterFixture_snapshotIncludesRouterConstraint(t *testing.T) {
	root := mustAbs(t, "testdata/router")
	files, err := collectFtFiles(root)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	snap, err := semantic.BuildSnapshot(files, root, nil)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	cat := snap.Types["catalog.Catalog"]
	var hasRouter bool
	for _, c := range cat.Constraints {
		if c.Name == "Router" {
			hasRouter = true
		}
	}
	if !hasRouter {
		t.Fatalf("catalog.Catalog constraints = %#v", cat.Constraints)
	}
}
