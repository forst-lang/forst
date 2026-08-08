package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Cross-file types must typecheck when merged (types.ft + consumer).
func TestGenerateCommand_directory_crossFileTypes_mergedTypecheck(t *testing.T) {
	dir := t.TempDir()
	mainDir := filepath.Join(dir, "main")
	if err := os.MkdirAll(mainDir, 0o755); err != nil {
		t.Fatal(err)
	}
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
	if err := os.WriteFile(filepath.Join(mainDir, "uses.ft"), []byte(usesSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "types.ft"), []byte(typesSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	types, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "types.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(types)
	if !strings.Contains(s, "R") {
		t.Fatalf("expected merged types with R; got:\n%s", s)
	}
	if strings.Contains(s, "export function GetX") {
		t.Fatalf("types.d.ts must not include function signatures; got:\n%s", s)
	}
	core, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "core", "main.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(core), "GetX") {
		t.Fatalf("expected GetX in core module; got:\n%s", core)
	}
}

// Merged multi-file package (no fixtures under examples/): discovery + generate must emit shared
// shapes and exported functions in one types.d.ts.
func TestGenerateCommand_mergedMultiFileSyntheticPackage(t *testing.T) {
	dir := t.TempDir()
	ftconfig := `{
  "compiler": {
    "target": "go",
    "optimization": "debug",
    "strict": false,
    "reportPhases": false,
    "reportMemoryUsage": false,
    "exportStructFields": true
  },
  "files": {
    "include": ["**/*.ft"],
    "exclude": ["**/node_modules/**"],
    "maxDepth": 10
  }
}
`
	typesSrc := `package main

type Catalog = {
	id: Int
}

type Order = {
	tag: String
}
`
	apiSrc := `package main

func GetId(a Catalog): Int {
	return a.id
}

func Tag(b Order): String {
	return b.tag
}
`
	for _, pair := range []struct {
		name string
		body string
	}{
		{"ftconfig.json", ftconfig},
	} {
		if err := os.WriteFile(filepath.Join(dir, pair.name), []byte(pair.body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mainDir := filepath.Join(dir, "main")
	if err := os.MkdirAll(mainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "types.ft"), []byte(typesSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "api.ft"), []byte(apiSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	typesPath := filepath.Join(defaultClientDistDir(dir), "types.d.ts")
	b, err := os.ReadFile(typesPath)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, needle := range []string{
		"Catalog",
		"Order",
	} {
		if !strings.Contains(s, needle) {
			t.Fatalf("generated types.d.ts missing %q; snippet:\n%s", needle, truncateForTestLog(s, 2000))
		}
	}
	core, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "core", "main.js"))
	if err != nil {
		t.Fatal(err)
	}
	coreText := string(core)
	for _, needle := range []string{"GetId", "Tag"} {
		if !strings.Contains(coreText, needle) {
			t.Fatalf("core module missing %q; snippet:\n%s", needle, truncateForTestLog(coreText, 2000))
		}
	}
}

func truncateForTestLog(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "…"
}
