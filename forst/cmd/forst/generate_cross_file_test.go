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

type Alpha = {
	id: Int
}

type Beta = {
	tag: String
}
`
	apiSrc := `package main

func GetId(a Alpha): Int {
	return a.id
}

func Tag(b Beta): String {
	return b.tag
}
`
	for _, pair := range []struct {
		name string
		body string
	}{
		{"ftconfig.json", ftconfig},
		{"types.ft", typesSrc},
		{"api.ft", apiSrc},
	} {
		if err := os.WriteFile(filepath.Join(dir, pair.name), []byte(pair.body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	typesPath := filepath.Join(dir, "generated", "types.d.ts")
	b, err := os.ReadFile(typesPath)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, needle := range []string{
		"Alpha",
		"Beta",
		"GetId",
		"Tag",
	} {
		if !strings.Contains(s, needle) {
			t.Fatalf("generated types.d.ts missing %q; snippet:\n%s", needle, truncateForTestLog(s, 2000))
		}
	}
}

func truncateForTestLog(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "…"
}
