package main

import (
	"path/filepath"
	"testing"
)

func TestCollectFtPaths_SkipsNonFt(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFmtTestFile(t, dir, "x.go", "package x\n", 0o600)

	got, err := collectFtPaths([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestCollectFtPaths_FileInputAndMissingPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ftPath := writeFmtTestFile(t, dir, "single.ft", "package main\n", 0o600)
	txtPath := writeFmtTestFile(t, dir, "single.txt", "x", 0o600)

	gotFt, err := collectFtPaths([]string{ftPath})
	if err != nil {
		t.Fatalf("collectFtPaths(ft): %v", err)
	}
	if len(gotFt) != 1 || gotFt[0] != ftPath {
		t.Fatalf("unexpected ft file result: %v", gotFt)
	}

	gotTxt, err := collectFtPaths([]string{txtPath})
	if err != nil {
		t.Fatalf("collectFtPaths(txt): %v", err)
	}
	if len(gotTxt) != 0 {
		t.Fatalf("expected no files for non-ft input, got %v", gotTxt)
	}

	_, err = collectFtPaths([]string{filepath.Join(dir, "does-not-exist.ft")})
	if err == nil {
		t.Fatal("expected stat error for missing path")
	}
}
