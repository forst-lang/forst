package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// generateTSExamplesManifest is examples/in/forst-generate-ts-examples.json (paths relative to examples/in).
// Add an entry when an example ships ftconfig + .ft sources and should pass `forst generate` + tsc.
type generateTSExamplesManifest struct {
	Examples []struct {
		Path        string   `json:"path"`
		MustContain []string `json:"mustContain,omitempty"`
	} `json:"examples"`
}

// examplesInRoot returns the absolute path to examples/in (from forst/cmd/forst test cwd).
func examplesInRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	base := filepath.Dir(thisFile) // forst/cmd/forst
	abs := filepath.Clean(filepath.Join(base, "..", "..", "..", "examples", "in"))
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("examples/in path %s: %v", abs, err)
	}
	return abs
}

func TestGenerate_typescriptTypechecks_exampleManifest(t *testing.T) {
	root := examplesInRoot(t)
	manifestPath := filepath.Join(root, "forst-generate-ts-examples.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest %s: %v (add examples/in/forst-generate-ts-examples.json to register examples)", manifestPath, err)
	}
	var man generateTSExamplesManifest
	if err := json.Unmarshal(data, &man); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if len(man.Examples) == 0 {
		t.Fatal("manifest has no examples")
	}
	for _, ex := range man.Examples {
		ex := ex
		name := ex.Path
		if name == "" {
			t.Fatal("manifest entry missing path")
		}
		t.Run(name, func(t *testing.T) {
			srcDir := filepath.Join(root, filepath.Clean(name))
			if st, err := os.Stat(srcDir); err != nil || !st.IsDir() {
				t.Fatalf("example dir %s: %v", srcDir, err)
			}
			ftconfig := filepath.Join(srcDir, "ftconfig.json")
			if _, err := os.Stat(ftconfig); err != nil {
				t.Fatalf("example %q must have ftconfig.json: %v", name, err)
			}

			dir := t.TempDir()
			if err := copyGenerateExampleSources(srcDir, dir); err != nil {
				t.Fatalf("copy sources: %v", err)
			}
			if err := generateCommand([]string{dir}); err != nil {
				t.Fatalf("generateCommand: %v", err)
			}
			if len(ex.MustContain) > 0 {
				mustContainInGeneratedTypeScript(t, dir, ex.MustContain)
			}
			assertTypeScriptCompiles(t, dir)
		})
	}
}

// copyGenerateExampleSources copies .ft files and ftconfig.json from an example tree, skipping
// generated output dirs so the temp project matches a clean checkout.
func copyGenerateExampleSources(srcRoot, dstRoot string) error {
	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if base == "generated" || base == "client" || base == "node_modules" || base == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		base := filepath.Base(path)
		if !strings.HasSuffix(base, ".ft") && base != "ftconfig.json" {
			return nil
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(dstRoot, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0644)
	})
}

func mustContainInGeneratedTypeScript(t *testing.T, projectRoot string, want []string) {
	t.Helper()
	var combined strings.Builder
	for _, sub := range []string{"generated", "client"} {
		dir := filepath.Join(projectRoot, sub)
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if strings.HasSuffix(path, ".ts") {
				b, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				combined.Write(b)
				combined.WriteByte('\n')
			}
			return nil
		})
	}
	s := combined.String()
	for _, w := range want {
		if !strings.Contains(s, w) {
			t.Fatalf("generated TypeScript must contain %q (manifest mustContain). Output snippet:\n%s",
				w, truncateForTestLog(s, 4000))
		}
	}
}
