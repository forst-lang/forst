package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tictactoeExampleDir is examples/in/tictactoe relative to forst/cmd/forst (package main tests).
const tictactoeExampleDir = "../../../examples/in/tictactoe"

// Cross-file types must typecheck when merged (types.ft + consumer).
func TestGenerateCommand_directory_crossFileTypes_mergedTypecheck(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(dir, "uses.ft"), []byte(usesSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "types.ft"), []byte(typesSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	types, err := os.ReadFile(filepath.Join(dir, "generated", "types.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(types)
	if !strings.Contains(s, "R") || !strings.Contains(s, "GetX") {
		t.Fatalf("expected merged types with R and GetX; got:\n%s", s)
	}
}

// Real multi-file package (types + engine + server): merged typecheck + generate must emit shared shapes.
func TestGenerateCommand_tictactoeMergedPackage(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"types.ft", "engine.ft", "server.ft", "ftconfig.json"} {
		src := filepath.Join(tictactoeExampleDir, name)
		b, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read %s: %v", src, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	typesPath := filepath.Join(dir, "generated", "types.d.ts")
	types, err := os.ReadFile(typesPath)
	if err != nil {
		t.Fatal(err)
	}
	s := string(types)
	for _, needle := range []string{
		"GameState",
		"MoveRequest",
		"MoveResponse",
		"PlayMove",
		"NewGame",
		"ApplyMove",
	} {
		if !strings.Contains(s, needle) {
			t.Fatalf("generated types.d.ts missing %q; snippet:\n%s", needle, truncateForTestLog(s, 2000))
		}
	}
}

func truncateForTestLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
