package devcompile

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/forstpkg"
	"forst/internal/modulecheck"
)

func TestSession_ParseFile_cachesByFingerprint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(path, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewSession(dir)
	n1, err := s.ParseFile(nil, path)
	if err != nil {
		t.Fatal(err)
	}
	n2, err := s.ParseFile(nil, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(n1) != len(n2) {
		t.Fatalf("cache should return same nodes: %d vs %d", len(n1), len(n2))
	}
	s.NoteChange(path)
	n3, err := s.ParseFile(nil, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(n3) != len(n1) {
		t.Fatalf("expected reparsed nodes after invalidation")
	}
}

func TestSession_CachedModuleResult_invalidatesOnFileEdit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewSession(dir)
	modResult := &modulecheck.ModuleResult{
		ModuleRoot:      dir,
		ForstPkgToFiles: map[string][]string{"main": {path}},
	}
	s.StoreModuleResult(dir, modResult)
	if _, ok := s.CachedModuleResult(dir); !ok {
		t.Fatal("expected cached module result")
	}
	s.NoteChange(path)
	if _, ok := s.CachedModuleResult(dir); !ok {
		t.Fatal("expected cache hit when file unchanged on disk after NoteChange")
	}
	if err := os.WriteFile(path, []byte("package main\n// edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.CachedModuleResult(dir); ok {
		t.Fatal("expected cache miss after file edit on disk")
	}
}

func TestCollectForstImportLocals(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.ft")
	if err := os.WriteFile(path, []byte("package api\nimport auth \"providers/auth\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nodes, err := forstpkg.ParseForstFile(nil, path)
	if err != nil {
		t.Fatal(err)
	}
	locals := CollectForstImportLocals(nodes)
	if len(locals) != 1 || locals[0] != "auth" {
		t.Fatalf("locals=%v want [auth]", locals)
	}
}

func TestSession_ParseAndMerge_deterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.ft")
	b := filepath.Join(dir, "b.ft")
	if err := os.WriteFile(a, []byte("package main\nfunc A() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("package main\nfunc B() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewSession(dir)
	merged, _, err := s.ParseAndMerge(nil, []string{b, a})
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) < 2 {
		t.Fatalf("expected merged nodes, got %d", len(merged))
	}
}

func TestModuleFingerprint_changesOnEdit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r1 := &modulecheck.ModuleResult{ForstPkgToFiles: map[string][]string{"main": {path}}}
	fp1 := moduleFingerprint(r1)
	if err := os.WriteFile(path, []byte("package main\n// edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fp2 := moduleFingerprint(r1)
	if fp1 == fp2 {
		t.Fatal("fingerprint should change when file content changes")
	}
}
