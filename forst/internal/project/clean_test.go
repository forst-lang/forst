package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanGenerated_removesDotForst(t *testing.T) {
	dir := t.TempDir()
	dotForst := filepath.Join(dir, GeneratedDirName)
	for _, sub := range []string{"run/s1", "gen/test/t1", "exec/1/pkg/fn"} {
		if err := os.MkdirAll(filepath.Join(dotForst, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dotForst, "invoke.ready"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := CleanGenerated(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Removed) != 1 || result.Removed[0] != dotForst {
		t.Fatalf("Removed = %v want [%s]", result.Removed, dotForst)
	}
	if _, err := os.Stat(dotForst); !os.IsNotExist(err) {
		t.Fatalf(".forst still exists: %v", err)
	}
}

func TestCleanGenerated_missingDir_isNoop(t *testing.T) {
	dir := t.TempDir()
	result, err := CleanGenerated(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Removed) != 0 {
		t.Fatalf("expected no removals, got %v", result.Removed)
	}
}

func TestCleanGenerated_dryRun(t *testing.T) {
	dir := t.TempDir()
	dotForst := filepath.Join(dir, GeneratedDirName)
	if err := os.MkdirAll(dotForst, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := CleanGenerated(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if !result.DryRun || len(result.Removed) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, err := os.Stat(dotForst); err != nil {
		t.Fatalf(".forst should still exist after dry-run: %v", err)
	}
}

func TestCleanGenerated_preservesForstGomod(t *testing.T) {
	dir := t.TempDir()
	forstGomod := filepath.Join(dir, ".forst-gomod")
	if err := os.MkdirAll(forstGomod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forstGomod, "go.mod"), []byte("module example.com/app/forst\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, GeneratedDirName, "run"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := CleanGenerated(dir, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(forstGomod); err != nil {
		t.Fatalf(".forst-gomod removed: %v", err)
	}
}
