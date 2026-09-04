package testutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testutil"
)

func TestListFixtureDirs_empty(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "fixtures")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	fx, err := testutil.ListFixtureDirs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(fx) != 0 {
		t.Fatalf("want 0, got %d", len(fx))
	}
}

func TestLoadFixtureDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), []byte(`{"id":"demo","expect":"parse-ok"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "input.ft"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fx, err := testutil.LoadFixtureDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fx.Meta.ID != "demo" || fx.Meta.Expect != "parse-ok" {
		t.Fatalf("meta=%+v", fx.Meta)
	}
	if fx.Src != "package main\n" {
		t.Fatalf("src=%q", fx.Src)
	}
}
