package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forst/internal/ftconfig"
	transformerts "forst/internal/transformer/ts"

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

func writeMainFt(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// Smoke: valid shared fixture typechecks and produces outDir/dist/types.d.ts (no tsc required).
func TestGenerateCommand_minimalFixture_generatesTypes(t *testing.T) {
	dir := t.TempDir()
	ftPath := writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{ftPath}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	requireGenerateOutputForTSC(t, dir, minimalEchoFixtureTypeScriptChecks)
}

func TestGenerateCommand_generateStreamingClientsFlag_doesNotBreakGenerate(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "ftconfig.json")
	if err := os.WriteFile(cfgPath, []byte(`{"compiler":{"generateStreamingClients":true}}`), 0644); err != nil {
		t.Fatal(err)
	}
	ftPath := writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{ftPath}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	clientPath := filepath.Join(defaultClientDistDir(dir), "core", "main.js")
	data, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("read client: %v", err)
	}
	s := string(data)
	// Echo returns a shape, not chan T — no *Stream method without a typable channel row type.
	if strings.Contains(s, "invokeStream<") {
		t.Fatalf("did not expect streaming emission for non-channel return, got:\n%s", s)
	}
}

func TestGenerateCommand_unknownShapeFieldTypeFails(t *testing.T) {
	dir := t.TempDir()
	ftPath := writeMainFt(t, dir, `package main

type EchoRequest = {
	message: Stringd
}

func Echo(input EchoRequest) {
	return { echo: input.message, timestamp: 0 }
}
`)
	if err := generateCommand([]string{ftPath}); err == nil {
		t.Fatal("expected error: unknown type Stringd in shape field")
	}
}

func TestGenerateCommand_publicWithProvidersFailsSidecarExport(t *testing.T) {
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "main.ft")
	src := `package main

type Logger = { info(msg String) }

func PublicApi() {
	use logger: Logger
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := generateCommand([]string{ftPath})
	if err == nil {
		t.Fatal("expected sidecar export error for public function with Providers")
	}
	if !strings.Contains(err.Error(), "cannot export PublicApi") {
		t.Fatalf("unexpected error: %v", err)
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

func TestGenerateCommand_logsResolvedGenerateConfig(t *testing.T) {
	dir := t.TempDir()
	writeMainFt(t, dir, generateTestMinimalValidForst)

	var buf bytes.Buffer
	prev := newGenerateLogger
	t.Cleanup(func() { newGenerateLogger = prev })
	newGenerateLogger = func() *logrus.Logger {
		log := logrus.New()
		log.SetLevel(logrus.InfoLevel)
		log.SetOutput(&buf)
		log.SetFormatter(&logrus.TextFormatter{DisableColors: true, DisableTimestamp: true})
		return log
	}

	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Resolved generate config") {
		t.Fatalf("expected resolved config log, got:\n%s", out)
	}
	if !strings.Contains(out, "packageName="+ftconfig.DefaultPackageName) &&
		!strings.Contains(out, `packageName="`+ftconfig.DefaultPackageName+`"`) {
		t.Fatalf("expected packageName %s in log, got:\n%s", ftconfig.DefaultPackageName, out)
	}
	if !strings.Contains(out, "outDir=.forst/client") &&
		!strings.Contains(out, `outDir=".forst/client"`) {
		t.Fatalf("expected default outDir in log, got:\n%s", out)
	}
}

func TestGenerateCommand_singleFtFileWritesSelfContainedClient(t *testing.T) {
	dir := t.TempDir()
	ftPath := writeMainFt(t, dir, generateTestMinimalValidForst)

	if err := generateCommand([]string{ftPath}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}

	outDir := defaultClientOutDir(dir)
	for _, rel := range []string{
		"dist/types.d.ts",
		"dist/core/main.js",
		"dist/pkg/main.js",
		"dist/index.js",
		"dist/transport.js",
		"package.json",
		"README.md",
	} {
		path := filepath.Join(outDir, rel)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s: %v", rel, err)
		}
	}

	types, err := os.ReadFile(filepath.Join(outDir, "dist", "types.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(types), "EchoRequest") {
		t.Fatalf("types.d.ts should mention EchoRequest; got:\n%s", types)
	}
	if strings.Contains(string(types), "export function Echo(") {
		t.Fatalf("types must be shapes only; function signatures live on package modules, not types.d.ts:\n%s", types)
	}

	client, err := os.ReadFile(filepath.Join(outDir, "dist", "core", "main.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(client), "invokeFunction") {
		t.Fatalf("client module should use invokeFunction; got:\n%s", client)
	}
	if !strings.Contains(string(client), "../transport.js") {
		t.Fatalf("core module should import ../transport.js; got:\n%s", client)
	}
	if strings.Contains(string(client), "@forst/client") {
		t.Fatalf("client module must not import @forst/client; got:\n%s", client)
	}
	if !strings.Contains(string(client), "export const main") {
		t.Fatalf("client export should match package name main; got:\n%s", client)
	}
	if !strings.Contains(string(client), "getDefaultInvokeClient") || !strings.Contains(string(client), "../transport.js") {
		t.Fatalf("generated core should import transport, got:\n%s", client)
	}
	dts, err := os.ReadFile(filepath.Join(outDir, "dist", "core", "main.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dts), "EchoRequest") || !strings.Contains(string(dts), `from "../types.js"`) {
		t.Fatalf("core.d.ts should import types from ../types.js, got:\n%s", dts)
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
	ftPath := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(ftPath, []byte("not valid forst {{{"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{ftPath}); err == nil {
		t.Fatal("expected error when the only file fails to parse/transform")
	}
	if _, err := os.Stat(filepath.Join(defaultClientDistDir(dir), "types.d.ts")); err == nil {
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
	good := writeMainFt(t, dir, generateTestMinimalValidForst)
	ignored := filepath.Join(dir, "ignored.ft")
	if err := os.WriteFile(ignored, []byte(generateTestSecondForstFile), 0644); err != nil {
		t.Fatal(err)
	}
	_ = good
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	types, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "types.d.ts"))
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
	if err := os.WriteFile(blocked, []byte("package blocked\n"), 0644); err != nil {
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
	mainDir := filepath.Join(dir, "main")
	if err := os.MkdirAll(mainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "echo.ft"), []byte(generateTestMinimalValidForst), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "ping.ft"), []byte(generateTestSecondForstFile), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	srcDir := defaultClientDistDir(dir)
	types, err := os.ReadFile(filepath.Join(srcDir, "types.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(types)
	if !strings.Contains(s, "EchoRequest") || !strings.Contains(s, "Ping") {
		t.Fatalf("merged types.d.ts should include shapes from both files; got:\n%s", s)
	}
	if strings.Contains(s, "export function") {
		t.Fatalf("types must be shapes only; function signatures live on package modules, not types.d.ts:\n%s", s)
	}
	core, err := os.ReadFile(filepath.Join(srcDir, "core", "main.js"))
	if err != nil {
		t.Fatal(err)
	}
	coreText := string(core)
	if !strings.Contains(coreText, "Echo") || !strings.Contains(coreText, "PingServer") {
		t.Fatalf("core module should include both functions; got:\n%s", coreText)
	}
	if _, err := os.Stat(filepath.Join(srcDir, "pkg", "main.js")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(srcDir, "pkg", "a.js")); err == nil {
		t.Fatal("expected no per-file stem clients")
	}
}

func testClientPackageOutputs(pkg string, fnNames ...string) []*transformerts.TypeScriptOutput {
	fns := make([]transformerts.FunctionSignature, len(fnNames))
	for i, name := range fnNames {
		fns[i] = transformerts.FunctionSignature{Name: name, ReturnType: "unknown"}
	}
	return []*transformerts.TypeScriptOutput{{PackageName: pkg, SourceFileStem: pkg, Functions: fns}}
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
	ft := writeMainFt(t, dir, generateTestMinimalValidForst)
	badCfg := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(badCfg, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := generateCommand([]string{"-config", badCfg, ft})
	if err == nil || !strings.Contains(err.Error(), "load config") {
		t.Fatalf("got %v", err)
	}
}

func TestGenerateCommand_mkdirSrcFails(t *testing.T) {
	dir := t.TempDir()
	ft := writeMainFt(t, dir, generateTestMinimalValidForst)
	orig := generateIO.MkdirAll
	generateIO.MkdirAll = func(string, os.FileMode) error { return fmt.Errorf("mkdir") }
	t.Cleanup(func() { generateIO.MkdirAll = orig })
	err := generateCommand([]string{ft})
	if err == nil || (!strings.Contains(err.Error(), "dist/core directory") && !strings.Contains(err.Error(), "dist/pkg directory")) {
		t.Fatalf("got %v", err)
	}
}

func TestGenerateCommand_writeTypesFails(t *testing.T) {
	dir := t.TempDir()
	ft := writeMainFt(t, dir, generateTestMinimalValidForst)
	orig := generateIO.WriteFile
	generateIO.WriteFile = func(name string, data []byte, perm os.FileMode) error {
		if strings.Contains(filepath.Base(name), "types.d.ts") && strings.Contains(name, filepath.Join(".forst", "client", "dist")) {
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
	ft := writeMainFt(t, dir, generateTestMinimalValidForst)
	orig := generateIO.WriteFile
	generateIO.WriteFile = func(name string, data []byte, perm os.FileMode) error {
		if strings.Contains(name, filepath.Join("dist", "core")) && strings.Contains(filepath.Base(name), "main.js") {
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
	genCfg := ftconfig.EffectiveGenerateConfig(nil, "")
	err := generateClientPackage(t.TempDir(), genCfg, testClientPackageOutputs("a"), "6321", log, nil)
	if err == nil || !strings.Contains(err.Error(), "dist directory") {
		t.Fatalf("got %v", err)
	}
}

func TestGenerateClientPackage_writeIndexFails(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	dir := t.TempDir()
	origW := generateIO.WriteFile
	generateIO.WriteFile = func(name string, data []byte, perm os.FileMode) error {
		if strings.Contains(filepath.Base(name), "index.js") {
			return fmt.Errorf("index")
		}
		return origW(name, data, perm)
	}
	t.Cleanup(func() { generateIO.WriteFile = origW })
	genCfg := ftconfig.EffectiveGenerateConfig(nil, "")
	err := generateClientPackage(dir, genCfg, testClientPackageOutputs("a"), "6321", log, nil)
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
		if strings.Contains(filepath.Base(name), "package.json") {
			return fmt.Errorf("pj")
		}
		return origW(name, data, perm)
	}
	t.Cleanup(func() { generateIO.WriteFile = origW })
	genCfg := ftconfig.EffectiveGenerateConfig(nil, "")
	err := generateClientPackage(dir, genCfg, testClientPackageOutputs("a"), "6321", log, nil)
	if err == nil || !strings.Contains(err.Error(), "package.json") {
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
	ft := writeMainFt(t, dir, generateTestMinimalValidForst)
	orig := generateTSOutputsByPackageHook
	generateTSOutputsByPackageHook = func([]string, *logrus.Logger, *transformerts.GenerateTSOptions) ([]*transformerts.TypeScriptOutput, error) {
		return nil, fmt.Errorf("per package")
	}
	t.Cleanup(func() { generateTSOutputsByPackageHook = orig })
	err := generateCommand([]string{ft})
	if err == nil || !strings.Contains(err.Error(), "per package") {
		t.Fatalf("got %v", err)
	}
}

func TestGenerateCommand_mergeOutputsError(t *testing.T) {
	dir := t.TempDir()
	ft := writeMainFt(t, dir, generateTestMinimalValidForst)
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

func TestGenerateCommand_generateClientPackageReturnsError(t *testing.T) {
	dir := t.TempDir()
	ft := writeMainFt(t, dir, generateTestMinimalValidForst)
	orig := generateClientPackageHook
	generateClientPackageHook = func(string, ftconfig.GenerateConfig, []*transformerts.TypeScriptOutput, string, *logrus.Logger, *generateWriteStats) error {
		return fmt.Errorf("client")
	}
	t.Cleanup(func() { generateClientPackageHook = orig })
	err := generateCommand([]string{ft})
	if err == nil || !strings.Contains(err.Error(), "client") {
		t.Fatalf("expected generate client package error, got %v", err)
	}
}

func TestGenerateCommand_stemPackageMismatch_errorsByDefault(t *testing.T) {
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "foo.ft")
	src := `package bar

func Hello() {
	return { ok: true }
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	err := generateCommand([]string{ftPath})
	if err == nil {
		t.Fatal("expected stem/package mismatch error")
	}
	if !strings.Contains(err.Error(), "must match declared package") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateCommand_mergedPackageMain_requiresFlagUnlessAllowed(t *testing.T) {
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
	err := generateCommand([]string{dir})
	if err == nil {
		t.Fatal("expected stem/package mismatch error for merged package main without flag")
	}
	if !strings.Contains(err.Error(), "must match declared package") {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := generateCommand([]string{"-allow-stem-package-mismatch", dir}); err != nil {
		t.Fatalf("generateCommand with flag: %v", err)
	}
	if _, err := os.Stat(filepath.Join(defaultClientDistDir(dir), "pkg", "main.js")); err != nil {
		t.Fatalf("expected pkg/main.ts: %v", err)
	}
}

func TestGenerateCommand_allowStemPackageMismatch_generatesPackageClient(t *testing.T) {
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "bcrypt.ft")
	src := `package bcrypt

type HashRequest = {
	password: String
}

func Hash(input HashRequest) {
	return { digest: input.password }
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{"-allow-stem-package-mismatch", ftPath}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	if _, err := os.Stat(filepath.Join(defaultClientDistDir(dir), "pkg", "bcrypt.js")); err != nil {
		t.Fatalf("expected pkg/bcrypt.ts: %v", err)
	}
}

func TestGenerateCommand_multiPackage_bcryptAndMain(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.ft"), []byte(generateTestMinimalValidForst), 0644); err != nil {
		t.Fatal(err)
	}
	bcryptSrc := `package bcrypt

type HashRequest = {
	password: String
}

func Hash(input HashRequest) {
	return { digest: input.password }
}
`
	if err := os.WriteFile(filepath.Join(dir, "bcrypt.ft"), []byte(bcryptSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{"-allow-stem-package-mismatch", dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	srcDir := defaultClientDistDir(dir)
	for _, pkg := range []string{"main", "bcrypt"} {
		if _, err := os.Stat(filepath.Join(srcDir, "pkg", pkg+".js")); err != nil {
			t.Fatalf("missing pkg/%s.ts: %v", pkg, err)
		}
		if _, err := os.Stat(filepath.Join(srcDir, "core", pkg+".js")); err != nil {
			t.Fatalf("missing core/%s.ts: %v", pkg, err)
		}
	}
	idx, err := os.ReadFile(filepath.Join(srcDir, "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(idx), "bcrypt") || !strings.Contains(string(idx), "main") {
		t.Fatalf("client index should reference both packages:\n%s", idx)
	}
}

func TestGenerateCommand_multiFileSamePackage_singleMergedClient(t *testing.T) {
	dir := t.TempDir()
	authDir := filepath.Join(dir, "auth")
	if err := os.MkdirAll(authDir, 0o755); err != nil {
		t.Fatal(err)
	}
	typesSrc := `package auth

type Session = {
	token: String
}
`
	apiSrc := `package auth

func Login(input Session) {
	return { ok: true }
}
`
	if err := os.WriteFile(filepath.Join(authDir, "session.ft"), []byte(typesSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(authDir, "login.ft"), []byte(apiSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	srcDir := defaultClientDistDir(dir)
	if _, err := os.Stat(filepath.Join(srcDir, "pkg", "auth.js")); err != nil {
		t.Fatalf("expected single pkg/auth.ts: %v", err)
	}
	if _, err := os.Stat(filepath.Join(srcDir, "pkg", "login.js")); !os.IsNotExist(err) {
		t.Fatalf("expected no pkg/login.ts, stat err=%v", err)
	}
	types, err := os.ReadFile(filepath.Join(srcDir, "types.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(types)
	if !strings.Contains(s, "Session") {
		t.Fatalf("merged types should include Session:\n%s", s)
	}
	core, err := os.ReadFile(filepath.Join(srcDir, "core", "auth.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(core), "Login") {
		t.Fatalf("core module should include Login:\n%s", core)
	}
}

func TestGenerateCommand_prunesStaleClientWhenPackageRemoved(t *testing.T) {
	dir := t.TempDir()
	bcryptSrc := `package bcrypt

func Hash(input { password: String }) {
	return { digest: input.password }
}
`
	if err := os.WriteFile(filepath.Join(dir, "bcrypt.ft"), []byte(bcryptSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{"-allow-stem-package-mismatch", dir}); err != nil {
		t.Fatalf("first generate: %v", err)
	}
	srcDir := defaultClientDistDir(dir)
	staleFlat := filepath.Join(srcDir, "legacy.js")
	stalePkg := filepath.Join(srcDir, "pkg", "legacy.js")
	staleCore := filepath.Join(srcDir, "core", "legacy.js")
	for _, stale := range []string{staleFlat, stalePkg, staleCore} {
		if err := os.WriteFile(stale, []byte("// stale\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := generateCommand([]string{"-allow-stem-package-mismatch", dir}); err != nil {
		t.Fatalf("second generate: %v", err)
	}
	for _, stale := range []string{staleFlat, stalePkg, staleCore} {
		if _, err := os.Stat(stale); !os.IsNotExist(err) {
			t.Fatalf("expected stale %s pruned, stat err=%v", stale, err)
		}
	}
}

func TestGenerateCommand_skipsTypeOnlyPackageClient(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "models.ft"), []byte(`package models

type Item = {
	id: Int
}
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.ft"), []byte(generateTestMinimalValidForst), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	srcDir := defaultClientDistDir(dir)
	if _, err := os.Stat(filepath.Join(srcDir, "pkg", "models.js")); !os.IsNotExist(err) {
		t.Fatalf("type-only models package should not emit client, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(srcDir, "pkg", "main.js")); err != nil {
		t.Fatal(err)
	}
	types, err := os.ReadFile(filepath.Join(srcDir, "types.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(types), "Item") {
		t.Fatalf("types.d.ts should still include models.Item")
	}
}
