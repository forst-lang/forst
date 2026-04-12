package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Shared Forst sources for generate tests, dev server tests, and generate_tsc_test.go.
// Unknown type names in shapes (e.g. Stringd instead of String) fail typechecking, not generate.
const generateTestMinimalValidForst = `package main

type EchoRequest = {
	message: String
}

func Echo(input EchoRequest) {
	return {
		echo: input.message,
		timestamp: 1234567890,
	}
}
`

const generateTestSecondForstFile = `package main

type Ping = {
	ok: Bool
}

func PingServer(input Ping) {
	return { pong: input.ok }
}
`

// Smoke: valid shared fixture typechecks and produces generated/types.d.ts (no tsc required).
func TestGenerateCommand_minimalFixture_generatesTypes(t *testing.T) {
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "sample.ft")
	if err := os.WriteFile(ftPath, []byte(generateTestMinimalValidForst), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{ftPath}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	requireGenerateOutputForTSC(t, dir, minimalEchoFixtureTypeScriptChecks)
}

func TestGenerateCommand_unknownShapeFieldTypeFails(t *testing.T) {
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "bad.ft")
	src := `package main

type EchoRequest = {
	message: Stringd
}

func Echo(input EchoRequest) {
	return { echo: input.message, timestamp: 0 }
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{ftPath}); err == nil {
		t.Fatal("expected error: unknown type Stringd in shape field")
	}
}

func TestGenerateCommand_requiresTarget(t *testing.T) {
	err := generateCommand(nil)
	if err == nil {
		t.Fatal("expected error when args empty")
	}
	if !strings.Contains(err.Error(), "requires a target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateCommand_statError(t *testing.T) {
	err := generateCommand([]string{filepath.Join(t.TempDir(), "nonexistent.ft")})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "failed to stat") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateCommand_rejectsNonFtFile(t *testing.T) {
	tmp := t.TempDir()
	plain := filepath.Join(tmp, "readme.txt")
	if err := os.WriteFile(plain, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	err := generateCommand([]string{plain})
	if err == nil {
		t.Fatal("expected error for non-.ft file")
	}
	if !strings.Contains(err.Error(), ".ft extension") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateCommand_emptyDirectoryHasNoFtFiles(t *testing.T) {
	dir := t.TempDir()
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("expected nil when no .ft files, got %v", err)
	}
}

func TestGenerateCommand_singleFtFileWritesGeneratedAndClient(t *testing.T) {
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "sample.ft")
	if err := os.WriteFile(ftPath, []byte(generateTestMinimalValidForst), 0644); err != nil {
		t.Fatal(err)
	}

	if err := generateCommand([]string{ftPath}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}

	for _, rel := range []string{
		"generated/types.d.ts",
		"generated/sample.client.ts",
		"client/index.ts",
		"client/package.json",
		"client/types.d.ts",
	} {
		path := filepath.Join(dir, rel)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s: %v", rel, err)
		}
	}

	types, err := os.ReadFile(filepath.Join(dir, "generated", "types.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(types), "Echo") {
		t.Fatalf("types.d.ts should mention Echo; got:\n%s", types)
	}

	client, err := os.ReadFile(filepath.Join(dir, "generated", "sample.client.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(client), "invokeFunction") {
		t.Fatalf("client module should use invokeFunction; got:\n%s", client)
	}
	if !strings.Contains(string(client), "export const sample") {
		t.Fatalf("client export should match .ft stem sample; got:\n%s", client)
	}
	if !strings.Contains(string(client), "import type {") || !strings.Contains(string(client), "EchoRequest") || !strings.Contains(string(client), "from './types'") {
		t.Fatalf("generated client should import types from ./types, got:\n%s", client)
	}
	if strings.Contains(string(client), "export interface EchoRequest") {
		t.Fatalf("generated client should not duplicate interfaces from types.d.ts, got:\n%s", client)
	}
}

func TestFindForstFiles_nestedAndFlat(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "nested")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		filepath.Join(root, "a.ft"),
		filepath.Join(sub, "b.ft"),
	} {
		if err := os.WriteFile(p, []byte("package main\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := DefaultConfig()
	files, err := cfg.FindForstFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("want 2 .ft files, got %d: %v", len(files), files)
	}
}

func TestGenerateCommand_invalidForstFile_returnsErrorAndNoGeneratedArtifacts(t *testing.T) {
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "bad.ft")
	if err := os.WriteFile(ftPath, []byte("not valid forst {{{"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{ftPath}); err == nil {
		t.Fatal("expected error when the only file fails to parse/transform")
	}
	if _, err := os.Stat(filepath.Join(dir, "generated", "types.d.ts")); err == nil {
		t.Fatal("expected no types.d.ts when generation fails")
	}
}

func TestGenerateCommand_respectsFtconfigExclude(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "ftconfig.json")
	cfgJSON := `{
  "files": {
    "include": ["**/*.ft"],
    "exclude": ["**/ignored.ft"]
  }
}`
	if err := os.WriteFile(cfgPath, []byte(cfgJSON), 0644); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(dir, "good.ft")
	if err := os.WriteFile(good, []byte(generateTestMinimalValidForst), 0644); err != nil {
		t.Fatal(err)
	}
	ignored := filepath.Join(dir, "ignored.ft")
	if err := os.WriteFile(ignored, []byte(generateTestSecondForstFile), 0644); err != nil {
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
	if !strings.Contains(s, "Echo") {
		t.Fatalf("expected Echo from good.ft; got:\n%s", s)
	}
	if strings.Contains(s, "PingServer") {
		t.Fatalf("ignored.ft should be excluded; got PingServer in types:\n%s", s)
	}
}

func TestGenerateCommand_singleExcludedFile_returnsError(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "ftconfig.json")
	cfgJSON := `{
  "files": {
    "include": ["**/*.ft"],
    "exclude": ["**/blocked.ft"]
  }
}`
	if err := os.WriteFile(cfgPath, []byte(cfgJSON), 0644); err != nil {
		t.Fatal(err)
	}
	blocked := filepath.Join(dir, "blocked.ft")
	if err := os.WriteFile(blocked, []byte(generateTestMinimalValidForst), 0644); err != nil {
		t.Fatal(err)
	}
	err := generateCommand([]string{blocked})
	if err == nil {
		t.Fatal("expected error when single file is excluded by ftconfig")
	}
	if !strings.Contains(err.Error(), "not included") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateCommand_directoryMergesTypesIntoSingleTypesDotDts(t *testing.T) {
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
	types, err := os.ReadFile(filepath.Join(dir, "generated", "types.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(types)
	if !strings.Contains(s, "Echo") || !strings.Contains(s, "PingServer") {
		t.Fatalf("merged types.d.ts should include both files; got:\n%s", s)
	}
	if _, err := os.Stat(filepath.Join(dir, "generated", "a.client.ts")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "generated", "b.client.ts")); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateClientIndex_importsPackages(t *testing.T) {
	idx := generateClientIndex([]string{"alpha", "beta"})
	for _, frag := range []string{
		"import { alpha } from '../generated/alpha.client'",
		"import { beta } from '../generated/beta.client'",
		"public alpha:",
		"public beta:",
		"export type * from './types.d.ts'",
	} {
		if !strings.Contains(idx, frag) {
			t.Fatalf("missing %q in index:\n%s", frag, idx)
		}
	}
}

func TestGenerateClientPackageJson_isValidJSONShape(t *testing.T) {
	j := generateClientPackageJSON()
	if !strings.Contains(j, `"name": "@forst/client"`) || !strings.Contains(j, `"@forst/sidecar"`) {
		t.Fatalf("unexpected package.json:\n%s", j)
	}
}

func TestCopyFile_roundTrip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	payload := []byte("hello generate")
	if err := os.WriteFile(src, payload, 0644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("got %q, want %q", got, payload)
	}
}

func TestCopyFile_missingSource(t *testing.T) {
	err := copyFile(filepath.Join(t.TempDir(), "nope"), filepath.Join(t.TempDir(), "out"))
	if err == nil {
		t.Fatal("expected error")
	}
}
