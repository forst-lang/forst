package discovery

import (
	"path/filepath"
	"testing"
)

func TestBuildForstPackageImportPaths(t *testing.T) {
	root := filepath.Clean("/proj")
	paths := map[string][]string{
		"alpha": {filepath.Join(root, "alpha", "alpha.ft")},
		"beta":  {filepath.Join(root, "beta", "beta.ft")},
	}
	got := BuildForstPackageImportPaths(root, "example.com/app", paths)
	if got["example.com/app/alpha"] != "alpha" {
		t.Fatalf("alpha path: %v", got)
	}
	if got["example.com/app/beta"] != "beta" {
		t.Fatalf("beta path: %v", got)
	}
}
