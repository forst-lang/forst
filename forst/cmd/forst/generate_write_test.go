package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateWrite_skipsIdenticalBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.js")
	content := []byte("export const x = 1;\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	var stats generateWriteStats
	if err := writeGeneratedFile(path, content, &stats); err != nil {
		t.Fatalf("writeGeneratedFile: %v", err)
	}
	if stats.Written != 0 || stats.Skipped != 1 {
		t.Fatalf("stats = %+v, want Written=0 Skipped=1", stats)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("file changed on skip: got %q", got)
	}
}

func TestGenerateWrite_writesWhenContentChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.js")
	if err := os.WriteFile(path, []byte("old\n"), 0644); err != nil {
		t.Fatal(err)
	}

	next := []byte("new\n")
	var stats generateWriteStats
	if err := writeGeneratedFile(path, next, &stats); err != nil {
		t.Fatalf("writeGeneratedFile: %v", err)
	}
	if stats.Written != 1 || stats.Skipped != 0 {
		t.Fatalf("stats = %+v, want Written=1 Skipped=0", stats)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, next) {
		t.Fatalf("got %q, want %q", got, next)
	}
}

func TestGenerateWrite_isAtomicOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.js")
	content := []byte("atomic\n")

	var renameCalls [][2]string
	origRename := generateIO.Rename
	origWrite := generateIO.WriteFile
	t.Cleanup(func() {
		generateIO.Rename = origRename
		generateIO.WriteFile = origWrite
	})

	generateIO.WriteFile = func(name string, data []byte, perm os.FileMode) error {
		if name == path {
			t.Fatalf("WriteFile wrote target path directly: %s", name)
		}
		if !strings.HasPrefix(filepath.Base(name), filepath.Base(path)+".tmp-") {
			t.Fatalf("WriteFile temp name %q not under %s.tmp-*", name, filepath.Base(path))
		}
		if filepath.Dir(name) != filepath.Dir(path) {
			t.Fatalf("temp file dir %q != target dir %q", filepath.Dir(name), filepath.Dir(path))
		}
		return origWrite(name, data, perm)
	}
	generateIO.Rename = func(oldpath, newpath string) error {
		renameCalls = append(renameCalls, [2]string{oldpath, newpath})
		return origRename(oldpath, newpath)
	}

	var stats generateWriteStats
	if err := writeGeneratedFile(path, content, &stats); err != nil {
		t.Fatalf("writeGeneratedFile: %v", err)
	}
	if stats.Written != 1 {
		t.Fatalf("stats.Written = %d, want 1", stats.Written)
	}
	if len(renameCalls) != 1 {
		t.Fatalf("Rename calls = %d, want 1", len(renameCalls))
	}
	if renameCalls[0][1] != path {
		t.Fatalf("Rename dest = %q, want %q", renameCalls[0][1], path)
	}
	if !strings.HasPrefix(filepath.Base(renameCalls[0][0]), filepath.Base(path)+".tmp-") {
		t.Fatalf("Rename source %q is not a temp sibling", renameCalls[0][0])
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Fatalf("partial temp file left behind: %s", e.Name())
		}
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("got %q, want %q", got, content)
	}
}

func TestGenerateWrite_byteStableAcrossTwoWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.js")
	content := []byte("stable\n")

	var stats1 generateWriteStats
	if err := writeGeneratedFile(path, content, &stats1); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if stats1.Written != 1 || stats1.Skipped != 0 {
		t.Fatalf("first stats = %+v, want Written=1 Skipped=0", stats1)
	}

	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var writeCount int
	origWrite := generateIO.WriteFile
	t.Cleanup(func() { generateIO.WriteFile = origWrite })
	generateIO.WriteFile = func(name string, data []byte, perm os.FileMode) error {
		writeCount++
		return origWrite(name, data, perm)
	}

	var stats2 generateWriteStats
	if err := writeGeneratedFile(path, content, &stats2); err != nil {
		t.Fatalf("second write: %v", err)
	}
	if stats2.Written != 0 || stats2.Skipped != 1 {
		t.Fatalf("second stats = %+v, want Written=0 Skipped=1", stats2)
	}
	if writeCount != 0 {
		t.Fatalf("second pass WriteFile calls = %d, want 0", writeCount)
	}

	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("bytes changed across identical writes:\nfirst=%q\nsecond=%q", first, second)
	}
}

func TestGenerateWrite_cleansTempWhenWriteFileFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.js")

	origWrite := generateIO.WriteFile
	t.Cleanup(func() { generateIO.WriteFile = origWrite })
	generateIO.WriteFile = func(name string, data []byte, perm os.FileMode) error {
		if strings.HasPrefix(filepath.Base(name), filepath.Base(path)+".tmp-") {
			return fmt.Errorf("no space left on device")
		}
		return origWrite(name, data, perm)
	}

	err := writeGeneratedFile(path, []byte("x\n"), nil)
	if err == nil {
		t.Fatal("expected WriteFile error")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Fatalf("temp file left after failed WriteFile: %s", e.Name())
		}
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("target should not exist after failed WriteFile, stat err=%v", err)
	}
}

func TestGenerateWrite_cleansTempWhenRenameFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.js")

	origRename := generateIO.Rename
	t.Cleanup(func() { generateIO.Rename = origRename })
	generateIO.Rename = func(oldpath, newpath string) error {
		return fmt.Errorf("rename blocked")
	}

	err := writeGeneratedFile(path, []byte("x\n"), nil)
	if err == nil {
		t.Fatal("expected rename error")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Fatalf("temp file left after failed rename: %s", e.Name())
		}
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("target should not exist after failed rename, stat err=%v", err)
	}
}
