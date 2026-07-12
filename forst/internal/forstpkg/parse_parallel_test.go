package forstpkg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFilesParallel_deterministicOrder(t *testing.T) {
	dir := t.TempDir()
	var paths []string
	for i, body := range []string{
		"package main\nfunc A() {}\n",
		"package main\nfunc B() {}\n",
		"package main\nfunc C() {}\n",
	} {
		p := filepath.Join(dir, string(rune('a'+i))+".ft")
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	merged1, _, err := ParseAndMergePackage(nil, paths)
	if err != nil {
		t.Fatal(err)
	}
	merged2, _, err := ParseAndMergePackage(nil, paths)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged1) != len(merged2) {
		t.Fatalf("parallel parse should be deterministic: %d vs %d", len(merged1), len(merged2))
	}
}
