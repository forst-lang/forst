package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

func ensureNodeModulesDir(t *testing.T, projectRoot string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(projectRoot, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	linkErrorsPackage(t, projectRoot)
}

func TestGenerate_emitsDistFiles(t *testing.T) {
	dir := t.TempDir()
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	outDir := defaultClientOutDir(dir)
	for _, rel := range []string{
		"dist/index.js",
		"dist/index.d.ts",
		"dist/errors.js",
		"dist/errors.d.ts",
		"dist/transport.js",
		"dist/transport.d.ts",
		"dist/types.d.ts",
		"dist/core/main.js",
		"dist/core/main.d.ts",
		"dist/pkg/main.js",
		"dist/pkg/main.d.ts",
		"package.json",
	} {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(outDir, "src")); !os.IsNotExist(err) {
		t.Fatalf("src/ must not be written, stat err=%v", err)
	}
}

func TestGenerate_emitsNoCjsFiles(t *testing.T) {
	dir := t.TempDir()
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	err := filepath.WalkDir(defaultClientOutDir(dir), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.HasSuffix(path, ".cjs") {
			t.Fatalf("unexpected .cjs file: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGenerate_userModulesEmittedUnderDistPkgAndDistCore(t *testing.T) {
	dir := t.TempDir()
	writeTwoPackageProject(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	dist := defaultClientDistDir(dir)
	for _, rel := range []string{
		"pkg/alpha.js", "pkg/alpha.d.ts", "core/alpha.js", "core/alpha.d.ts",
		"pkg/beta.js", "pkg/beta.d.ts", "core/beta.js", "core/beta.d.ts",
	} {
		if _, err := os.Stat(filepath.Join(dist, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
}

func TestGenerate_packageJSONExportsPointToDistPkg(t *testing.T) {
	j := generateClientPackageJSON(ftconfig.EffectiveGenerateConfig(nil, ""), []string{"auth"}, nil)
	var pkg map[string]any
	if err := json.Unmarshal([]byte(j), &pkg); err != nil {
		t.Fatal(err)
	}
	exports := pkg["exports"].(map[string]any)
	entry := exports["./auth"].(map[string]any)
	if entry["types"] != "./dist/pkg/auth.d.ts" {
		t.Fatalf("types = %#v", entry["types"])
	}
	if entry["default"] != "./dist/pkg/auth.js" {
		t.Fatalf("default = %#v", entry["default"])
	}
	root := exports["."].(map[string]any)
	if root["types"] != "./dist/index.d.ts" || root["default"] != "./dist/index.js" {
		t.Fatalf("root exports = %#v", root)
	}
}

func TestGenerate_packageJSONDeclaresNodeEnginesFloor(t *testing.T) {
	j := generateClientPackageJSON(ftconfig.EffectiveGenerateConfig(nil, ""), nil, nil)
	var pkg map[string]any
	if err := json.Unmarshal([]byte(j), &pkg); err != nil {
		t.Fatal(err)
	}
	engines, ok := pkg["engines"].(map[string]any)
	if !ok {
		t.Fatalf("missing engines:\n%s", j)
	}
	if engines["node"] != ">=20.19" {
		t.Fatalf("engines.node = %#v, want >=20.19", engines["node"])
	}
}

func TestGenerate_headerCommentUsesInvokePort(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "ftconfig.json")
	if err := os.WriteFile(cfgPath, []byte(`{"server":{"port":"9999"},"generate":{"link":"never"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	core, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "core", "main.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(core), "http://127.0.0.1:9999") {
		t.Fatalf("expected port 9999 in header:\n%s", core)
	}
	if strings.Contains(string(core), "8081") {
		t.Fatalf("must not hardcode 8081:\n%s", core)
	}
}

func TestWriteFileAtomic_skipsWriteWhenBytesMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.js")
	content := []byte("same\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	written, err := writeFileAtomic(path, content)
	if err != nil {
		t.Fatal(err)
	}
	if written {
		t.Fatal("expected skip when bytes match")
	}
}

func TestWriteFileAtomic_neverLeavesPartialFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.js")
	origRename := generateIO.Rename
	t.Cleanup(func() { generateIO.Rename = origRename })
	generateIO.Rename = func(_, _ string) error {
		return os.ErrPermission
	}
	_, err := writeFileAtomic(path, []byte("partial\n"))
	if err == nil {
		t.Fatal("expected rename failure")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Fatalf("temp left behind: %s", e.Name())
		}
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("target should not exist after failed write, err=%v", err)
	}
}

func TestGenerate_isByteStableAcrossRuns(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"generate":{"link":"never"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("first generate: %v", err)
	}
	first := readDistTree(t, defaultClientOutDir(dir))
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("second generate: %v", err)
	}
	second := readDistTree(t, defaultClientOutDir(dir))
	if len(first) != len(second) {
		t.Fatalf("file count changed: %d -> %d", len(first), len(second))
	}
	for path, a := range first {
		b, ok := second[path]
		if !ok {
			t.Fatalf("missing on second run: %s", path)
		}
		if a != b {
			t.Fatalf("bytes changed for %s", path)
		}
	}
}

func TestGenerate_secondRunRewritesNothingWhenSourcesUnchanged(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"generate":{"link":"never"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("first generate: %v", err)
	}

	var writeCount int
	origWrite := generateIO.WriteFile
	t.Cleanup(func() { generateIO.WriteFile = origWrite })
	generateIO.WriteFile = func(name string, data []byte, perm os.FileMode) error {
		writeCount++
		return origWrite(name, data, perm)
	}
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("second generate: %v", err)
	}
	if writeCount != 0 {
		t.Fatalf("second run WriteFile calls = %d, want 0", writeCount)
	}
}

func TestGenerate_watchRegeneratesOnFtChange(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"generate":{"link":"never"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	ftPath := writeMainFt(t, dir, generateTestMinimalValidForst)

	kick := make(chan struct{}, 1)
	prevWatch := watchPackageRootFn
	prevDebounce := generateWatchDebounce
	prevStop := generateWatchStopHook
	generateWatchDebounce = 5 * time.Millisecond
	stopHook := make(chan struct{})
	generateWatchStopHook = stopHook
	watchPackageRootFn = func(_ *logrus.Logger, _ string, _ *ftconfig.Config, _ time.Duration, onChange func(changedPath string), stop <-chan struct{}) error {
		select {
		case <-kick:
			onChange(ftPath)
		case <-stop:
			return nil
		}
		<-stop
		return nil
	}
	stopped := false
	t.Cleanup(func() {
		watchPackageRootFn = prevWatch
		generateWatchDebounce = prevDebounce
		generateWatchStopHook = prevStop
		if !stopped {
			close(stopHook)
		}
	})

	done := make(chan error, 1)
	go func() {
		done <- generateCommand([]string{"-watch", dir})
	}()

	deadline := time.Now().Add(5 * time.Second)
	corePath := filepath.Join(defaultClientDistDir(dir), "core", "main.js")
	for {
		if _, err := os.Stat(corePath); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for initial generate")
		}
		time.Sleep(10 * time.Millisecond)
	}

	newSrc := `package main

type EchoRequest = {
	message: String
}

func EchoRenamed(input EchoRequest) {
	return { message: input.message }
}
`
	if err := os.WriteFile(ftPath, []byte(newSrc), 0644); err != nil {
		t.Fatal(err)
	}
	kick <- struct{}{}

	deadline = time.Now().Add(5 * time.Second)
	var got []byte
	var err error
	for {
		got, err = os.ReadFile(corePath)
		if err == nil && strings.Contains(string(got), "EchoRenamed") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for watch regenerate; last:\n%s", got)
		}
		time.Sleep(10 * time.Millisecond)
	}

	close(stopHook)
	stopped = true
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("watch exited with error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("watch did not stop after stop hook")
	}
}

func TestGenerate_neverWritesLegacyDirectories(t *testing.T) {
	src, err := os.ReadFile("generate.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	for _, banned := range []string{`"generated"`, `"client"`} {
		if strings.Contains(text, banned) {
			t.Fatalf("generate.go must not use %s as an output path segment", banned)
		}
	}
}

func TestGenerate_tsc_resolvesThroughNodeModulesLink(t *testing.T) {
	if os.Getenv("FORST_SKIP_TS_E2E") == "1" {
		t.Skip("FORST_SKIP_TS_E2E=1")
	}
	dir := t.TempDir()
	ensureNodeModulesDir(t, dir)
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	link := filepath.Join(dir, "node_modules", "@forst", "gen")
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("expected node_modules/@forst/gen link: %v", err)
	}
	// tsconfig has no paths; resolution must use the link.
	assertTypeScriptCompiles(t, dir)
}

func TestGenerate_tsc_requireFromCommonJsResolves(t *testing.T) {
	if os.Getenv("FORST_SKIP_TS_E2E") == "1" {
		t.Skip("FORST_SKIP_TS_E2E=1")
	}
	dir := t.TempDir()
	ensureNodeModulesDir(t, dir)
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	cjs := filepath.Join(dir, "require-smoke.cjs")
	body := `
const path = require("node:path");
const mod = require("@forst/gen/main");
if (typeof mod.Echo !== "function") {
  console.error("Echo missing", Object.keys(mod));
  process.exit(1);
}
console.log("ok");
`
	if err := os.WriteFile(cjs, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	// Node must be new enough for require(esm); skip cleanly otherwise.
	out, err := runNodeRequireSmoke(t, dir, cjs)
	if err != nil {
		blob := err.Error() + "\n" + out
		if strings.Contains(blob, "ERR_REQUIRE_ESM") || strings.Contains(blob, "not supported") {
			t.Skipf("Node require(esm) unavailable: %s", blob)
		}
		t.Fatalf("require smoke failed: %v\n%s", err, out)
	}
}

func runNodeRequireSmoke(t *testing.T, dir, script string) (string, error) {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found")
		return "", nil
	}
	cmd := exec.Command("node", script)
	cmd.Dir = dir
	b, err := cmd.CombinedOutput()
	return string(b), err
}

func readDistTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		// Skip marker absolute boundaryRoot which embeds host paths.
		if filepath.Base(path) == ".forst-generated" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
