package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// effectGenerateGoldenFiles are dist modules whose Effect-mode emit is snapshotted under testdata/effect/golden/.
var effectGenerateGoldenFiles = []string{
	"errors.js",
	"errors.d.ts",
	"domain-errors.js",
	"domain-errors.d.ts",
}

func effectGenerateGoldenDir() string {
	return filepath.Join("testdata", "effect", "golden")
}

func updateEffectGenerateGoldens(t *testing.T) {
	t.Helper()
	dist := generateEffectProject(t, t.TempDir())
	goldenDir := effectGenerateGoldenDir()
	if err := os.MkdirAll(goldenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, rel := range effectGenerateGoldenFiles {
		src := filepath.Join(dist, rel)
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read generated %s: %v", rel, err)
		}
		dst := filepath.Join(goldenDir, rel)
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			t.Fatalf("write golden %s: %v", rel, err)
		}
		t.Logf("wrote golden %s", dst)
	}
}

// TestGenerate_effectMode_matchesCommittedGoldens compares Effect-mode error module emit
// against committed snapshots under testdata/effect/golden/.
// Regenerate via TestUpdateExamplesGoldens / task examples:update-goldens (UPDATE_EXAMPLES_GOLDENS=1).
func TestGenerate_effectMode_matchesCommittedGoldens(t *testing.T) {
	if os.Getenv("UPDATE_EXAMPLES_GOLDENS") == "1" {
		t.Skip("golden update runs via TestUpdateExamplesGoldens")
	}

	dist := generateEffectProject(t, t.TempDir())
	goldenDir := effectGenerateGoldenDir()

	for _, rel := range effectGenerateGoldenFiles {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			goldenPath := filepath.Join(goldenDir, rel)
			expected, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden %s: %v (set UPDATE_EXAMPLES_GOLDENS=1 and run TestUpdateExamplesGoldens)", goldenPath, err)
			}
			actual, err := os.ReadFile(filepath.Join(dist, rel))
			if err != nil {
				t.Fatalf("read generated %s: %v", rel, err)
			}
			if string(expected) != string(actual) {
				t.Fatalf("golden mismatch for %s (set UPDATE_EXAMPLES_GOLDENS=1 and run TestUpdateExamplesGoldens)\n--- expected ---\n%s\n--- actual ---\n%s",
					rel, string(expected), string(actual))
			}
			for _, frag := range []string{`from "effect"`, "Data.TaggedError"} {
				if !strings.Contains(string(actual), frag) {
					t.Fatalf("effect golden %s missing %q", rel, frag)
				}
			}
		})
	}
}
