package discovery

import (
	"path/filepath"
	"testing"
)

func TestBuildForstPackageImportPaths(t *testing.T) {
	root := filepath.Clean("/proj")
	paths := map[string][]string{
		"catalog": {filepath.Join(root, "catalog", "catalog.ft")},
		"orders":  {filepath.Join(root, "orders", "orders.ft")},
	}
	got := BuildForstPackageImportPaths(root, "example.com/app", paths)
	if got["example.com/app/catalog"] != "catalog" {
		t.Fatalf("catalog path: %v", got)
	}
	if got["example.com/app/orders"] != "orders" {
		t.Fatalf("orders path: %v", got)
	}
}
