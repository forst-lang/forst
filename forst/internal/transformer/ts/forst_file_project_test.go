package transformerts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestParseMergedTypecheckProject_emptyInput(t *testing.T) {
	_, _, err := ParseMergedTypecheckProject(nil, logrus.New())
	if err == nil {
		t.Fatal("expected error for empty file list")
	}
}

// Two-file package: type in one file, function using it in another — must typecheck when merged.
func TestParseMergedTypecheckProject_twoFilesCrossReferences(t *testing.T) {
	dir := t.TempDir()
	typesSrc := `package main

type R = {
	x: Int
}
`
	usesSrc := `package main

func GetX(r R): Int {
	return r.x
}
`
	if err := os.WriteFile(filepath.Join(dir, "types.ft"), []byte(typesSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "uses.ft"), []byte(usesSrc), 0644); err != nil {
		t.Fatal(err)
	}
	paths := []string{
		filepath.Join(dir, "types.ft"),
		filepath.Join(dir, "uses.ft"),
	}
	chunks, tc, err := ParseMergedTypecheckProject(paths, logrus.New())
	if err != nil {
		t.Fatalf("ParseMergedTypecheckProject: %v", err)
	}
	if tc == nil {
		t.Fatal("expected typechecker")
	}
	if len(chunks) != 2 {
		t.Fatalf("chunks: got %d want 2", len(chunks))
	}
	if chunks[0].Stem != "types" || chunks[1].Stem != "uses" {
		t.Fatalf("stems: %#v %#v", chunks[0].Stem, chunks[1].Stem)
	}
	if len(chunks[0].Nodes) == 0 || len(chunks[1].Nodes) == 0 {
		t.Fatal("expected non-empty AST per file")
	}

	outputs, err := GenerateTypeScriptOutputsPerFile(chunks, tc, logrus.New(), nil)
	if err != nil {
		t.Fatalf("GenerateTypeScriptOutputsPerFile: %v", err)
	}
	if len(outputs) != 2 {
		t.Fatalf("TS outputs: got %d want 2", len(outputs))
	}
}
