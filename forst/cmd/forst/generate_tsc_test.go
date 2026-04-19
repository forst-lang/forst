package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed testdata/typescript/forst-sidecar.d.ts testdata/typescript/node-process-shim.d.ts
var tsE2EStubs embed.FS

func TestGenerate_typescriptTypechecks_singleFile(t *testing.T) {
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "sample.ft")
	if err := os.WriteFile(ftPath, []byte(generateTestMinimalValidForst), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{ftPath}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	requireGenerateOutputForTSC(t, dir, minimalEchoFixtureTypeScriptChecks)
	assertTypeScriptCompiles(t, dir)
}

func TestGenerate_typescriptTypechecks_mergedDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.ft"), []byte(generateTestMinimalValidForst), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.ft"), []byte(generateTestSecondForstFile), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	requireGenerateOutputForTSC(t, dir, mergedDirectoryFixtureTypeScriptChecks)
	assertTypeScriptCompiles(t, dir)
}

// minimalEchoFixtureTypeScriptChecks is shared with generate_test.go (same package).
// Shape-level correctness is enforced by the typechecker (unknown types are rejected); these
// are smoke checks that generate emitted expected declarations.
var minimalEchoFixtureTypeScriptChecks = []string{
	"export interface EchoRequest",
	"export function Echo(",
}

var mergedDirectoryFixtureTypeScriptChecks = []string{
	"export interface EchoRequest",
	"export function Echo(",
	"export interface Ping",
	"export function PingServer(",
}

// requireGenerateOutputForTSC fails if forst generate produced no usable output.
func requireGenerateOutputForTSC(t *testing.T, projectRoot string, wantSubstrings []string) {
	t.Helper()
	typesPath := filepath.Join(projectRoot, "generated", "types.d.ts")
	b, err := os.ReadFile(typesPath)
	if err != nil {
		t.Fatalf("expected generated/types.d.ts after generate (got error: %v). "+
			"If generate failed (parse/typecheck), generateCommand returns an error.", err)
	}
	typesSrc := string(b)
	for _, frag := range wantSubstrings {
		if !strings.Contains(typesSrc, frag) {
			t.Fatalf("generated/types.d.ts missing %q (run with -count=1 to avoid stale cache). Full file:\n%s",
				frag, typesSrc)
		}
	}
}

func assertTypeScriptCompiles(t *testing.T, projectRoot string) {
	t.Helper()
	if os.Getenv("FORST_SKIP_TS_E2E") == "1" {
		t.Skip("FORST_SKIP_TS_E2E=1")
	}
	if err := copyTSE2EStubs(projectRoot); err != nil {
		t.Fatalf("copy stubs: %v", err)
	}
	if err := writeTSConfig(projectRoot); err != nil {
		t.Fatalf("write tsconfig: %v", err)
	}
	if err := runTsc(t, projectRoot); err != nil {
		t.Fatal(err)
	}
}

func copyTSE2EStubs(projectRoot string) error {
	stubsDir := filepath.Join(projectRoot, "stubs")
	if err := os.MkdirAll(stubsDir, 0755); err != nil {
		return err
	}
	for _, name := range []string{"forst-sidecar.d.ts", "node-process-shim.d.ts"} {
		b, err := tsE2EStubs.ReadFile(filepath.Join("testdata/typescript", name))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(stubsDir, name), b, 0644); err != nil {
			return err
		}
	}
	return nil
}

type tsConfig struct {
	CompilerOptions tsCompilerOptions `json:"compilerOptions"`
	Include         []string          `json:"include"`
}

type tsCompilerOptions struct {
	Target           string `json:"target"`
	Module           string `json:"module"`
	ModuleResolution string `json:"moduleResolution"`
	Strict           bool   `json:"strict"`
	NoEmit           bool   `json:"noEmit"`
	SkipLibCheck     bool   `json:"skipLibCheck"`
	// No baseUrl: TS 6+ deprecates it (TS5101); paths resolve relative to the config file without it.
	Paths map[string][]string `json:"paths,omitempty"`
	// Empty types: do not auto-load @types/* from parent node_modules (avoids duplicate `process` vs node-process-shim.d.ts).
	Types []string `json:"types"`
}

func writeTSConfig(projectRoot string) error {
	cfg := tsConfig{
		CompilerOptions: tsCompilerOptions{
			Target:           "ES2022",
			Module:           "ESNext",
			ModuleResolution: "bundler",
			Strict:           true,
			NoEmit:           true,
			SkipLibCheck:     true,
			Types:            []string{},
			Paths: map[string][]string{
				"@forst/sidecar": {"./stubs/forst-sidecar.d.ts"},
			},
		},
		Include: []string{
			"generated/**/*.ts",
			"client/**/*.ts",
			"stubs/node-process-shim.d.ts",
		},
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(projectRoot, "tsconfig.json"), b, 0644)
}

func runTsc(t *testing.T, projectRoot string) error {
	t.Helper()
	tsconfigPath := filepath.Join(projectRoot, "tsconfig.json")

	if p, ok := findLocalTypescriptCompiler(); ok {
		cmd := exec.Command(p, "--noEmit", "-p", tsconfigPath)
		return runTscCmd(t, cmd)
	}

	repoRoot := findMonorepoRootWithPackageJSON()
	if repoRoot != "" {
		if _, err := exec.LookPath("bun"); err == nil {
			cmd := exec.Command("bun", "x", "tsc", "--noEmit", "-p", tsconfigPath)
			cmd.Dir = repoRoot
			return runTscCmd(t, cmd)
		}
	}

	if p, err := exec.LookPath("tsc"); err == nil {
		cmd := exec.Command(p, "--noEmit", "-p", tsconfigPath)
		return runTscCmd(t, cmd)
	}

	if _, err := exec.LookPath("npx"); err == nil {
		cmd := exec.Command("npx", "-y", "typescript@6.0.2", "tsc", "--noEmit", "-p", tsconfigPath)
		return runTscCmd(t, cmd)
	}

	t.Skip("no TypeScript compiler (install repo devDependencies with bun install, or install tsc / bun / npx)")
	return nil
}

func runTscCmd(t *testing.T, cmd *exec.Cmd) error {
	t.Helper()
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return &tscError{
			err:    err,
			stderr: stderr.String(),
			stdout: stdout.String(),
		}
	}
	return nil
}

type tscError struct {
	err    error
	stderr string
	stdout string
}

func (e *tscError) Error() string {
	if e.stderr != "" {
		return e.err.Error() + "\n" + e.stderr
	}
	if e.stdout != "" {
		return e.err.Error() + "\n" + e.stdout
	}
	return e.err.Error()
}

// findLocalTypescriptCompiler returns node_modules/typescript/bin/tsc walking up from cwd.
func findLocalTypescriptCompiler() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for i := 0; i < 16; i++ {
		binDir := filepath.Join(dir, "node_modules", "typescript", "bin")
		for _, name := range []string{"tsc", "tsc.cmd"} {
			cand := filepath.Join(binDir, name)
			if st, err := os.Stat(cand); err == nil && !st.IsDir() {
				return cand, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func findMonorepoRootWithPackageJSON() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for i := 0; i < 16; i++ {
		pkg := filepath.Join(dir, "package.json")
		if st, err := os.Stat(pkg); err == nil && !st.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
