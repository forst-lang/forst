package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	transformerts "forst/internal/transformer/ts"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
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

func TestCopyFile_writeDestinationFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not portable for read-only destination")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	if err := os.WriteFile(src, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	dstDir := filepath.Join(dir, "ro")
	if err := os.Mkdir(dstDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dstDir, 0o755) })
	dst := filepath.Join(dstDir, "dst.txt")
	if err := copyFile(src, dst); err == nil {
		t.Fatal("expected error when destination is not writable")
	}
}

func TestLoadConfigForGenerate_explicitPath_absError(t *testing.T) {
	_, err := loadConfigForGenerate("x\x00y", "dummy", false)
	if err == nil {
		t.Fatal("expected error from filepath.Abs on invalid path")
	}
}

func TestDiscoverForstFilesForGenerate_rejectsNonFtExtension(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "x.go")
	if err := os.WriteFile(f, []byte("package x"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	_, _, err := discoverForstFilesForGenerate(cfg, f, false)
	if err == nil || !strings.Contains(err.Error(), ".ft") {
		t.Fatalf("got %v", err)
	}
}

func TestDiscoverForstFilesForGenerate_directoryListsFtFiles(t *testing.T) {
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "one.ft")
	if err := os.WriteFile(ftPath, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	files, outDir, err := discoverForstFilesForGenerate(cfg, dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("want 1 file, got %v", files)
	}
	if filepath.Clean(outDir) != filepath.Clean(dir) {
		t.Fatalf("outputDir %q vs dir %q", outDir, dir)
	}
}

func TestLoadConfigForGenerate_explicitConfig_loadError(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := loadConfigForGenerate(bad, dir, true)
	if err == nil {
		t.Fatal("expected load config error")
	}
}

func TestDiscoverForstFilesForGenerate_directoryAbsError(t *testing.T) {
	cfg := DefaultConfig()
	_, _, err := discoverForstFilesForGenerate(cfg, "x\x00y", true)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDiscoverForstFilesForGenerate_missingDirTarget_findFails(t *testing.T) {
	cfg := DefaultConfig()
	missing := filepath.Join(t.TempDir(), "not-there")
	_, _, err := discoverForstFilesForGenerate(cfg, missing, true)
	if err == nil {
		t.Fatal("expected FindForstFiles error for missing directory")
	}
}

func TestDiscoverForstFilesForGenerate_directoryTarget_absError(t *testing.T) {
	orig := absPathForGenerate
	absPathForGenerate = func(string) (string, error) { return "", fmt.Errorf("abs") }
	t.Cleanup(func() { absPathForGenerate = orig })
	_, _, err := discoverForstFilesForGenerate(DefaultConfig(), t.TempDir(), true)
	if err == nil {
		t.Fatal("expected abs error")
	}
}

func TestDiscoverForstFilesForGenerate_fileTarget_absError(t *testing.T) {
	orig := absPathForGenerate
	absPathForGenerate = func(string) (string, error) { return "", fmt.Errorf("abs") }
	t.Cleanup(func() { absPathForGenerate = orig })
	_, _, err := discoverForstFilesForGenerate(DefaultConfig(), "x.ft", false)
	if err == nil {
		t.Fatal("expected abs error")
	}
}

func TestDiscoverForstFilesForGenerate_findForstFilesWalkError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod walk error not portable")
	}
	root := t.TempDir()
	sub := filepath.Join(root, "blocked")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.ft"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sub, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sub, 0o755) })
	cfg := DefaultConfig()
	_, _, err := discoverForstFilesForGenerate(cfg, root, true)
	if err == nil {
		t.Fatal("expected walk error")
	}
}

func TestDiscoverForstFilesForGenerate_singleFileAbsError(t *testing.T) {
	cfg := DefaultConfig()
	_, _, err := discoverForstFilesForGenerate(cfg, "a\x00b.ft", false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDiscoverForstFilesForGenerate_findInParentDirError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod walk error not portable")
	}
	dir := t.TempDir()
	ft := filepath.Join(dir, "one.ft")
	if err := os.WriteFile(ft, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	cfg := DefaultConfig()
	_, _, err := discoverForstFilesForGenerate(cfg, ft, false)
	if err == nil {
		t.Fatal("expected find error")
	}
}

func TestGenerateCommand_badFlag(t *testing.T) {
	err := generateCommand([]string{"-not-a-generate-flag"})
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestGenerateCommand_loadConfigFailure(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "x.ft")
	if err := os.WriteFile(ft, []byte(generateTestMinimalValidForst), 0o644); err != nil {
		t.Fatal(err)
	}
	badCfg := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(badCfg, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := generateCommand([]string{"-config", badCfg, ft})
	if err == nil || !strings.Contains(err.Error(), "load config") {
		t.Fatalf("got %v", err)
	}
}

func TestGenerateCommand_mkdirGeneratedFails(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "x.ft")
	if err := os.WriteFile(ft, []byte(generateTestMinimalValidForst), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := generateIO.MkdirAll
	generateIO.MkdirAll = func(string, os.FileMode) error { return fmt.Errorf("mkdir") }
	t.Cleanup(func() { generateIO.MkdirAll = orig })
	err := generateCommand([]string{ft})
	if err == nil || !strings.Contains(err.Error(), "generated") {
		t.Fatalf("got %v", err)
	}
}

func TestGenerateCommand_writeTypesFails(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "x.ft")
	if err := os.WriteFile(ft, []byte(generateTestMinimalValidForst), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := generateIO.WriteFile
	generateIO.WriteFile = func(name string, data []byte, perm os.FileMode) error {
		if strings.HasSuffix(name, "types.d.ts") && strings.Contains(name, "generated") {
			return fmt.Errorf("write types")
		}
		return orig(name, data, perm)
	}
	t.Cleanup(func() { generateIO.WriteFile = orig })
	err := generateCommand([]string{ft})
	if err == nil || !strings.Contains(err.Error(), "types declaration") {
		t.Fatalf("got %v", err)
	}
}

func TestGenerateCommand_writeClientModuleLogsError(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "x.ft")
	if err := os.WriteFile(ft, []byte(generateTestMinimalValidForst), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := generateIO.WriteFile
	generateIO.WriteFile = func(name string, data []byte, perm os.FileMode) error {
		if strings.HasSuffix(name, ".client.ts") {
			return fmt.Errorf("no client")
		}
		return orig(name, data, perm)
	}
	t.Cleanup(func() { generateIO.WriteFile = orig })
	if err := generateCommand([]string{ft}); err != nil {
		t.Fatalf("generateCommand completes with log+continue on client write: %v", err)
	}
}

func TestGenerateClientPackage_mkdirFails(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	orig := generateIO.MkdirAll
	generateIO.MkdirAll = func(string, os.FileMode) error { return fmt.Errorf("mkdir") }
	t.Cleanup(func() { generateIO.MkdirAll = orig })
	err := generateClientPackage(t.TempDir(), []string{"a"}, log)
	if err == nil || !strings.Contains(err.Error(), "client directory") {
		t.Fatalf("got %v", err)
	}
}

func TestGenerateClientPackage_writeIndexFails(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	dir := t.TempDir()
	origW := generateIO.WriteFile
	generateIO.WriteFile = func(name string, data []byte, perm os.FileMode) error {
		if strings.HasSuffix(name, "index.ts") {
			return fmt.Errorf("index")
		}
		return origW(name, data, perm)
	}
	t.Cleanup(func() { generateIO.WriteFile = origW })
	err := generateClientPackage(dir, []string{"a"}, log)
	if err == nil || !strings.Contains(err.Error(), "client index") {
		t.Fatalf("got %v", err)
	}
}

func TestGenerateClientPackage_writePackageJSONFails(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	dir := t.TempDir()
	origW := generateIO.WriteFile
	generateIO.WriteFile = func(name string, data []byte, perm os.FileMode) error {
		if strings.HasSuffix(name, "package.json") {
			return fmt.Errorf("pj")
		}
		return origW(name, data, perm)
	}
	t.Cleanup(func() { generateIO.WriteFile = origW })
	err := generateClientPackage(dir, []string{"a"}, log)
	if err == nil || !strings.Contains(err.Error(), "package.json") {
		t.Fatalf("got %v", err)
	}
}

func TestGenerateClientPackage_copyTypesFails(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	dir := t.TempDir()
	gen := filepath.Join(dir, "generated")
	if err := os.MkdirAll(gen, 0o755); err != nil {
		t.Fatal(err)
	}
	typesPath := filepath.Join(gen, "types.d.ts")
	if err := os.WriteFile(typesPath, []byte("export type X = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	origW := generateIO.WriteFile
	generateIO.WriteFile = func(name string, data []byte, perm os.FileMode) error {
		if strings.Contains(name, string(filepath.Separator)+"client"+string(filepath.Separator)) && strings.HasSuffix(name, "types.d.ts") {
			return fmt.Errorf("copy")
		}
		return origW(name, data, perm)
	}
	t.Cleanup(func() { generateIO.WriteFile = origW })
	err := generateClientPackage(dir, []string{"a"}, log)
	if err == nil || !strings.Contains(err.Error(), "copy types") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadConfigForGenerate_explicitAbsError(t *testing.T) {
	orig := absPathForGenerate
	absPathForGenerate = func(string) (string, error) { return "", fmt.Errorf("abs") }
	t.Cleanup(func() { absPathForGenerate = orig })
	_, err := loadConfigForGenerate("cfg.json", ".", true)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadConfigForGenerate_startDirAbsError(t *testing.T) {
	orig := absPathForGenerate
	absPathForGenerate = func(string) (string, error) { return "", fmt.Errorf("abs") }
	t.Cleanup(func() { absPathForGenerate = orig })
	_, err := loadConfigForGenerate("", t.TempDir(), true)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerateCommand_generateTSPerFileError(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "x.ft")
	if err := os.WriteFile(ft, []byte(generateTestMinimalValidForst), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := generateTSOutputsPerFileHook
	generateTSOutputsPerFileHook = func([]transformerts.ForstFileChunk, *typechecker.TypeChecker, *logrus.Logger) ([]*transformerts.TypeScriptOutput, error) {
		return nil, fmt.Errorf("per file")
	}
	t.Cleanup(func() { generateTSOutputsPerFileHook = orig })
	err := generateCommand([]string{ft})
	if err == nil || !strings.Contains(err.Error(), "per file") {
		t.Fatalf("got %v", err)
	}
}

func TestGenerateCommand_mergeOutputsError(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "x.ft")
	if err := os.WriteFile(ft, []byte(generateTestMinimalValidForst), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := mergeTypeScriptOutputsHook
	mergeTypeScriptOutputsHook = func([]*transformerts.TypeScriptOutput) (*transformerts.TypeScriptOutput, error) {
		return nil, fmt.Errorf("merge")
	}
	t.Cleanup(func() { mergeTypeScriptOutputsHook = orig })
	err := generateCommand([]string{ft})
	if err == nil || !strings.Contains(err.Error(), "merge") {
		t.Fatalf("got %v", err)
	}
}

func TestGenerateCommand_generateClientPackageLogsError(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "x.ft")
	if err := os.WriteFile(ft, []byte(generateTestMinimalValidForst), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := generateClientPackageHook
	generateClientPackageHook = func(string, []string, *logrus.Logger) error {
		return fmt.Errorf("client")
	}
	t.Cleanup(func() { generateClientPackageHook = orig })
	if err := generateCommand([]string{ft}); err != nil {
		t.Fatalf("expected nil (error is logged), got %v", err)
	}
}
