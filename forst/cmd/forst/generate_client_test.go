package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ftconfig"
)

// TestGenerate_omissionReport checks provider-gated omission reporting.
func TestGenerate_omissionReport(t *testing.T) {
	dir := t.TempDir()
	writeAuthProviderFixture(t, dir)
	buf := captureGenerateLogs(t)

	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "generate: omitted 2 functions (unsatisfied providers)") {
		t.Fatalf("missing summary in omission report:\n%s", out)
	}
	for _, fn := range []string{"auth.Login", "auth.Register"} {
		if !strings.Contains(out, fn) {
			t.Fatalf("missing %q in omission report:\n%s", fn, out)
		}
	}
	if !strings.Contains(out, "Logger") || !strings.Contains(out, "not satisfied") {
		t.Fatalf("missing provider reason in omission report:\n%s", out)
	}

	pkgJS, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "pkg", "auth.js"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(pkgJS), "Login") {
		t.Fatalf("Login must not be exported after provider omission:\n%s", pkgJS)
	}
}

const acceptanceBcryptFT = `package bcrypt

type ComparePasswordRequest = {
	password: String
	hash: String
}

type ComparePasswordResponse = {
	valid: Bool
}

func ComparePassword(input ComparePasswordRequest) {
	return { valid: true }
}
`

const acceptanceAuthFT = `package auth

type LoginRequest = {
	email: String
}

func Login(input LoginRequest) {
	return { ok: true }
}
`

func writeAcceptanceBcryptAuth(t *testing.T, dir string) {
	t.Helper()
	for _, pair := range []struct {
		pkg string
		src string
	}{
		{"bcrypt", acceptanceBcryptFT},
		{"auth", acceptanceAuthFT},
	} {
		pkgDir := filepath.Join(dir, pair.pkg)
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pkgDir, pair.pkg+".ft"), []byte(pair.src), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func writePkgFT(t *testing.T, dir, pkg, src string) {
	t.Helper()
	pkgDir := filepath.Join(dir, pkg)
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, pkg+".ft"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertNodeModulesLink(t *testing.T, dir, packageName string) string {
	t.Helper()
	link := filepath.Join(dir, "node_modules", filepath.FromSlash(packageName))
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("expected node_modules link %s: %v", link, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		// Junction / directory copy fallback is acceptable on some platforms.
		if !info.IsDir() {
			t.Fatalf("link path is neither symlink nor directory: %s mode=%v", link, info.Mode())
		}
	}
	return link
}

func readGeneratedPackageJSON(t *testing.T, dir string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(defaultClientOutDir(dir), "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pkg map[string]any
	if err := json.Unmarshal(raw, &pkg); err != nil {
		t.Fatalf("package.json: %v\n%s", err, raw)
	}
	return pkg
}

func TestGenerate_subpathFunctionImport(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found")
	}
	dir := t.TempDir()
	ensureNodeModulesDir(t, dir)
	writeAcceptanceBcryptAuth(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	assertNodeModulesLink(t, dir, "@forst/gen")

	pkgJS := filepath.Join(defaultClientDistDir(dir), "pkg", "bcrypt.js")
	data, err := os.ReadFile(pkgJS)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "ComparePassword") && !strings.Contains(string(data), `export * from "../core/bcrypt.js"`) {
		t.Fatalf("pkg/bcrypt.js missing ComparePassword re-export:\n%s", data)
	}

	script := filepath.Join(dir, "a01-smoke.mjs")
	body := `
import { ComparePassword } from "@forst/gen/bcrypt";
if (typeof ComparePassword !== "function") {
  console.error("ComparePassword missing", ComparePassword);
  process.exit(1);
}
console.log("ok");
`
	if err := os.WriteFile(script, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runNodeRequireSmoke(t, dir, script)
	if err != nil {
		t.Fatalf("subpath import smoke failed: %v\n%s", err, out)
	}

	if os.Getenv("FORST_SKIP_TS_E2E") != "1" {
		assertTypeScriptCompilesSmoke(t, dir, `import { ComparePassword } from "@forst/gen/bcrypt";
export const fn = ComparePassword;
`)
	}
}

func TestGenerate_subpathTypeImport(t *testing.T) {
	dir := t.TempDir()
	ensureNodeModulesDir(t, dir)
	writeAcceptanceBcryptAuth(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	dts, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "pkg", "bcrypt.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(dts)
	if !strings.Contains(text, "ComparePasswordRequest") || !strings.Contains(text, `from "../types.js"`) {
		t.Fatalf("pkg/bcrypt.d.ts must re-export ComparePasswordRequest:\n%s", text)
	}

	if os.Getenv("FORST_SKIP_TS_E2E") == "1" {
		t.Log("alias: structure asserted; tsc skipped (FORST_SKIP_TS_E2E=1)")
		return
	}
	assertTypeScriptCompilesSmoke(t, dir, `import type { ComparePasswordRequest } from "@forst/gen/bcrypt";
const _check: ComparePasswordRequest = { password: "x", hash: "y" };
export type Req = typeof _check;
`)
}

// assertTypeScriptCompilesSmoke typechecks a custom consumer file through the node_modules link.
func assertTypeScriptCompilesSmoke(t *testing.T, projectRoot, smoke string) {
	t.Helper()
	if os.Getenv("FORST_SKIP_TS_E2E") == "1" {
		t.Skip("FORST_SKIP_TS_E2E=1")
	}
	if err := copyTSE2EStubs(projectRoot); err != nil {
		t.Fatalf("copy stubs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "app-smoke.ts"), []byte(smoke), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := tsConfig{
		CompilerOptions: tsCompilerOptions{
			Target:           "ES2022",
			Module:           "ESNext",
			ModuleResolution: "bundler",
			Strict:           true,
			NoEmit:           true,
			SkipLibCheck:     true,
			Types:            []string{},
		},
		Include: []string{
			"app-smoke.ts",
			clientDistIncludeGlob(projectRoot),
			"stubs/node-process-shim.d.ts",
		},
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "tsconfig.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
	if err := runTsc(t, projectRoot); err != nil {
		t.Fatal(err)
	}
}

func TestGenerate_installablePackage(t *testing.T) {
	dir := t.TempDir()
	ensureNodeModulesDir(t, dir)
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	pkg := readGeneratedPackageJSON(t, dir)
	deps, _ := pkg["dependencies"].(map[string]any)
	if deps == nil || deps["@forst/errors"] == nil {
		t.Fatalf("generated package.json must declare @forst/errors dependency:\n%v", pkg)
	}
	if _, ok := pkg["devDependencies"]; ok {
		t.Fatalf("generated package.json must omit devDependencies:\n%v", pkg)
	}
	peers, _ := pkg["peerDependencies"].(map[string]any)
	if peers["@forst/cli"] == nil {
		t.Fatalf("expected optional @forst/cli peer:\n%v", pkg)
	}
	if _, hasEffect := peers["effect"]; hasEffect {
		t.Fatalf("promise mode must not declare effect peer:\n%v", pkg)
	}
	meta, _ := pkg["peerDependenciesMeta"].(map[string]any)
	cliMeta, _ := meta["@forst/cli"].(map[string]any)
	if cliMeta["optional"] != true {
		t.Fatalf("@forst/cli peer must be optional:\n%v", pkg)
	}
	link := assertNodeModulesLink(t, dir, "@forst/gen")
	outDir := defaultClientOutDir(dir)
	resolved, err := filepath.EvalSymlinks(link)
	if err == nil {
		want, _ := filepath.EvalSymlinks(outDir)
		if filepath.Clean(resolved) != filepath.Clean(want) {
			t.Fatalf("link resolves to %s, want %s", resolved, want)
		}
	}
	if _, err := exec.LookPath("node"); err != nil {
		return
	}
	script := filepath.Join(dir, "a03-resolve.mjs")
	body := `
import { createForstClient } from "@forst/gen";
import { Echo } from "@forst/gen/main";
if (typeof createForstClient !== "function" || typeof Echo !== "function") {
  console.error("exports missing", { createForstClient, Echo });
  process.exit(1);
}
console.log("ok");
`
	if err := os.WriteFile(script, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runNodeRequireSmoke(t, dir, script)
	if err != nil {
		t.Fatalf("resolve smoke failed: %v\n%s", err, out)
	}
}

func TestGenerate_linkRestoredByPostinstall(t *testing.T) {
	dir := t.TempDir()
	ensureNodeModulesDir(t, dir)
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("first generate: %v", err)
	}
	link := assertNodeModulesLink(t, dir, "@forst/gen")

	// Simulate npm ci wiping node_modules.
	if err := os.RemoveAll(filepath.Join(dir, "node_modules")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("link should be gone after node_modules delete, err=%v", err)
	}

	// Negative path: import fails without the link.
	if _, err := exec.LookPath("node"); err == nil {
		script := filepath.Join(dir, "a03b-missing.mjs")
		body := `import "@forst/gen/main";`
		if err := os.WriteFile(script, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
		out, err := runNodeRequireSmoke(t, dir, script)
		if err == nil {
			t.Fatal("expected import to fail without node_modules link")
		}
		blob := err.Error() + "\n" + out
		if !strings.Contains(blob, "ERR_MODULE_NOT_FOUND") && !strings.Contains(blob, "Cannot find package") && !strings.Contains(blob, "Cannot find module") {
			t.Fatalf("expected module-not-found style error, got:\n%s", blob)
		}
	}
	readme, err := os.ReadFile(filepath.Join(defaultClientOutDir(dir), "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "postinstall") || !strings.Contains(string(readme), "forst generate") {
		t.Fatalf("README must name postinstall restore path:\n%s", readme)
	}

	// Recreate node_modules and re-run generate (postinstall).
	ensureNodeModulesDir(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("postinstall generate: %v", err)
	}
	assertNodeModulesLink(t, dir, "@forst/gen")
	if _, err := exec.LookPath("node"); err != nil {
		return
	}
	script := filepath.Join(dir, "a03b-restored.mjs")
	body := `
import { Echo } from "@forst/gen/main";
if (typeof Echo !== "function") process.exit(1);
console.log("ok");
`
	if err := os.WriteFile(script, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runNodeRequireSmoke(t, dir, script)
	if err != nil {
		t.Fatalf("import after restore failed: %v\n%s", err, out)
	}
}

func TestGenerate_packageName(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"@acme/web","version":"1.0.0"}`), 0644); err != nil {
			t.Fatal(err)
		}
		writeMainFt(t, dir, generateTestMinimalValidForst)

		var buf bytes.Buffer
		prev := generateReportWriter
		t.Cleanup(func() { generateReportWriter = prev })
		generateReportWriter = &buf

		if err := generateCommand([]string{dir}); err != nil {
			t.Fatalf("generateCommand: %v", err)
		}
		pkg := readGeneratedPackageJSON(t, dir)
		if pkg["name"] != ftconfig.DefaultPackageName {
			t.Fatalf("name=%v, want %s", pkg["name"], ftconfig.DefaultPackageName)
		}
		out := buf.String()
		if !strings.Contains(out, "@forst/gen") || !strings.Contains(out, `import { Echo } from "@forst/gen/main"`) {
			t.Fatalf("stdout must name specifier and example import:\n%s", out)
		}
	})

	t.Run("adopter_rename", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"@acme/web","version":"1.0.0"}`), 0644); err != nil {
			t.Fatal(err)
		}
		writeMainFt(t, dir, generateTestMinimalValidForst)
		if err := generateCommand([]string{dir}); err != nil {
			t.Fatalf("generateCommand: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"@acmecorp/web","version":"1.0.0"}`), 0644); err != nil {
			t.Fatal(err)
		}
		if err := generateCommand([]string{dir}); err != nil {
			t.Fatalf("second generate: %v", err)
		}
		pkg := readGeneratedPackageJSON(t, dir)
		if pkg["name"] != ftconfig.DefaultPackageName {
			t.Fatalf("adopter rename must not change packageName: %v", pkg["name"])
		}
	})

	t.Run("spaced_directory", func(t *testing.T) {
		weird := filepath.Join(t.TempDir(), "My App")
		if err := os.MkdirAll(weird, 0o755); err != nil {
			t.Fatal(err)
		}
		writeMainFt(t, weird, generateTestMinimalValidForst)
		if err := generateCommand([]string{weird}); err != nil {
			t.Fatalf("generate in My App: %v", err)
		}
		pkg := readGeneratedPackageJSON(t, weird)
		if pkg["name"] != ftconfig.DefaultPackageName {
			t.Fatalf("directory name must not become packageName: %v", pkg["name"])
		}
		tree := readDistTree(t, defaultClientOutDir(weird))
		for path, body := range tree {
			if strings.Contains(body, "My App") {
				t.Fatalf("directory name leaked into %s", path)
			}
		}
	})

	t.Run("configured_packageName", func(t *testing.T) {
		custom := t.TempDir()
		if err := os.WriteFile(filepath.Join(custom, "ftconfig.json"), []byte(`{"generate":{"packageName":"@acme/api-client","link":"never"}}`), 0644); err != nil {
			t.Fatal(err)
		}
		writeMainFt(t, custom, generateTestMinimalValidForst)
		if err := generateCommand([]string{custom}); err != nil {
			t.Fatalf("custom packageName: %v", err)
		}
		pkg := readGeneratedPackageJSON(t, custom)
		if pkg["name"] != "@acme/api-client" {
			t.Fatalf("custom packageName not applied: %v", pkg["name"])
		}
	})
}

// Bundles a subpath import with esbuild without resolve.alias when esbuild is on PATH.
func TestGenerate_bundlerBuild(t *testing.T) {
	if _, err := exec.LookPath("esbuild"); err != nil {
		t.Skip("esbuild not on PATH")
	}
	dir := t.TempDir()
	ensureNodeModulesDir(t, dir)
	writeAcceptanceBcryptAuth(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	entry := filepath.Join(dir, "entry.mjs")
	if err := os.WriteFile(entry, []byte(`import { ComparePassword } from "@forst/gen/bcrypt";
if (typeof ComparePassword !== "function") process.exit(2);
`), 0644); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(dir, "bundle.mjs")
	cmd := exec.Command("esbuild", entry, "--bundle", "--outfile="+outFile, "--platform=node", "--format=esm")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("esbuild bundle: %v\n%s", err, out)
	}
	if _, err := os.Stat(outFile); err != nil {
		t.Fatalf("bundle output missing: %v", err)
	}
}

func TestGenerate_esmAndRequireBothResolve(t *testing.T) {
	t.Log("alias: also covered by TestGenerate_tsc_requireFromCommonJsResolves and TestGenerate_emitsNoCjsFiles")
	dir := t.TempDir()
	ensureNodeModulesDir(t, dir)
	writeAcceptanceBcryptAuth(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	err := filepath.WalkDir(defaultClientOutDir(dir), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".cjs") {
			t.Fatalf("unexpected .cjs file: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	pkg := readGeneratedPackageJSON(t, dir)
	engines, _ := pkg["engines"].(map[string]any)
	if engines == nil || engines["node"] != ">=20.19" {
		t.Fatalf("engines.node want >=20.19, got %#v", engines)
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found")
	}
	esm := filepath.Join(dir, "a06-esm.mjs")
	if err := os.WriteFile(esm, []byte(`
import { ComparePassword } from "@forst/gen/bcrypt";
if (typeof ComparePassword !== "function") process.exit(1);
console.log("ok");
`), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runNodeRequireSmoke(t, dir, esm)
	if err != nil {
		t.Fatalf("esm import failed: %v\n%s", err, out)
	}
	cjs := filepath.Join(dir, "a06-cjs.cjs")
	if err := os.WriteFile(cjs, []byte(`
const { ComparePassword } = require("@forst/gen/bcrypt");
if (typeof ComparePassword !== "function") process.exit(1);
console.log("ok");
`), 0644); err != nil {
		t.Fatal(err)
	}
	out, err = runNodeRequireSmoke(t, dir, cjs)
	if err != nil {
		blob := err.Error() + "\n" + out
		if strings.Contains(blob, "ERR_REQUIRE_ESM") || strings.Contains(blob, "not supported") {
			t.Skipf("Node require(esm) unavailable: %s", blob)
		}
		t.Fatalf("cjs require failed: %v\n%s", err, out)
	}
}

func TestGenerate_crossPackageNameCollisions(t *testing.T) {
	t.Run("duplicate_function_name", func(t *testing.T) {
		dir := t.TempDir()
		for _, pkg := range []string{"bcrypt", "crypto"} {
			src := `package ` + pkg + `

type HashRequest = {
	password: String
}

func Hash(input HashRequest) {
	return { digest: input.password }
}
`
			writePkgFT(t, dir, pkg, src)
		}
		if err := generateCommand([]string{dir}); err != nil {
			t.Fatalf("same Hash in two packages must succeed: %v", err)
		}
		for _, pkg := range []string{"bcrypt", "crypto"} {
			data, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "core", pkg+".js"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), "Hash") {
				t.Fatalf("%s missing Hash", pkg)
			}
		}
	})

	t.Run("conflicting_Config_type", func(t *testing.T) {
		conflict := t.TempDir()
		writePkgFT(t, conflict, "billing", `package billing

type Config = {
	x: Int
}

func Ping() {
	return 1
}
`)
		writePkgFT(t, conflict, "auth", `package auth

type Config = {
	y: String
}

func Ping() {
	return 1
}
`)
		err := generateCommand([]string{conflict})
		if err == nil {
			t.Fatal("expected conflicting Config type to fail generate")
		}
		msg := err.Error()
		for _, frag := range []string{"Config", "billing", "auth"} {
			if !strings.Contains(msg, frag) {
				t.Fatalf("conflict error missing %q:\n%s", frag, msg)
			}
		}
	})

	t.Run("identical_Address_merge", func(t *testing.T) {
		same := t.TempDir()
		addr := `type Address = {
	street: String
}
`
		writePkgFT(t, same, "alpha", "package alpha\n\n"+addr+"\nfunc A() {\n\treturn 1\n}\n")
		writePkgFT(t, same, "beta", "package beta\n\n"+addr+"\nfunc B() {\n\treturn 1\n}\n")
		if err := generateCommand([]string{same}); err != nil {
			t.Fatalf("identical Address must merge: %v", err)
		}
		types, err := os.ReadFile(filepath.Join(defaultClientDistDir(same), "types.d.ts"))
		if err != nil {
			t.Fatal(err)
		}
		if c := strings.Count(string(types), "export interface Address"); c != 1 {
			t.Fatalf("want exactly one Address declaration, got %d:\n%s", c, types)
		}
	})
}

func TestGenerate_reservedPackageNames(t *testing.T) {
	t.Run("near_reserved_names", func(t *testing.T) {
		dir := t.TempDir()
		ensureNodeModulesDir(t, dir)
		linkErrorsPackage(t, dir)
		for _, pkg := range []string{"types", "index", "transport"} {
			writePkgFT(t, dir, pkg, "package "+pkg+"\n\nfunc Ping() {\n\treturn 1\n}\n")
		}
		if err := generateCommand([]string{dir}); err != nil {
			t.Fatalf("near-reserved packages must succeed: %v", err)
		}
		for _, pkg := range []string{"types", "index", "transport"} {
			if _, err := os.Stat(filepath.Join(defaultClientDistDir(dir), "pkg", pkg+".js")); err != nil {
				t.Fatalf("missing pkg/%s.js: %v", pkg, err)
			}
		}
		if _, err := exec.LookPath("node"); err == nil {
			script := filepath.Join(dir, "a07b-types.mjs")
			if err := os.WriteFile(script, []byte(`
import { Ping } from "@forst/gen/types";
if (typeof Ping !== "function") process.exit(1);
console.log("ok");
`), 0644); err != nil {
				t.Fatal(err)
			}
			out, err := runNodeRequireSmoke(t, dir, script)
			if err != nil {
				t.Fatalf("@forst/gen/types import failed: %v\n%s", err, out)
			}
		}
	})

	t.Run("rejected_testing_package", func(t *testing.T) {
		bad := t.TempDir()
		writePkgFT(t, bad, "testing", "package testing\n\nfunc Ping() {\n\treturn 1\n}\n")
		err := generateCommand([]string{bad})
		if err == nil {
			t.Fatal("expected reserved package testing to fail")
		}
		msg := err.Error()
		for _, frag := range []string{"testing", "testingSubpath"} {
			if !strings.Contains(msg, frag) {
				t.Fatalf("reserved error missing %q:\n%s", frag, msg)
			}
		}
	})

	for _, pkg := range []string{"errors"} {
		pkg := pkg
		t.Run("rejected_"+pkg+"_package", func(t *testing.T) {
			bad := t.TempDir()
			writePkgFT(t, bad, pkg, fmt.Sprintf("package %s\n\nfunc Ping() {\n\treturn 1\n}\n", pkg))
			err := generateCommand([]string{bad})
			if err == nil {
				t.Fatalf("expected reserved package %q to fail", pkg)
			}
			if !strings.Contains(err.Error(), pkg) {
				t.Fatalf("reserved error should mention %q:\n%s", pkg, err.Error())
			}
		})
	}

	t.Run("testingSubpath_override", func(t *testing.T) {
		ok := t.TempDir()
		ensureNodeModulesDir(t, ok)
		if err := os.WriteFile(filepath.Join(ok, "ftconfig.json"), []byte(`{"generate":{"testingSubpath":"test-double"}}`), 0644); err != nil {
			t.Fatal(err)
		}
		writePkgFT(t, ok, "testing", "package testing\n\nfunc Ping() {\n\treturn 1\n}\n")
		if err := generateCommand([]string{ok}); err != nil {
			t.Fatalf("testing allowed when testingSubpath overridden: %v", err)
		}
		for _, rel := range []string{"pkg/testing.js", "test-double.js"} {
			if _, err := os.Stat(filepath.Join(defaultClientDistDir(ok), rel)); err != nil {
				t.Fatalf("missing %s: %v", rel, err)
			}
		}
		pkgJSON := readGeneratedPackageJSON(t, ok)
		exports, _ := pkgJSON["exports"].(map[string]any)
		if _, ok := exports["./testing"]; !ok {
			t.Fatalf("exports missing ./testing:\n%v", exports)
		}
		if _, ok := exports["./test-double"]; !ok {
			t.Fatalf("exports missing ./test-double:\n%v", exports)
		}
	})
}

func TestGenerate_forstPackageNamedClient(t *testing.T) {
	dir := t.TempDir()
	ensureNodeModulesDir(t, dir)
	writePkgFT(t, dir, "client", "package client\n\nfunc Ping() {\n\treturn 1\n}\n")
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generate with Forst package client: %v", err)
	}
	outDir := defaultClientOutDir(dir)
	if _, err := os.Stat(filepath.Join(outDir, "package.json")); err != nil {
		t.Fatalf(".forst/client must hold generated package: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "dist", "pkg", "client.js")); err != nil {
		t.Fatalf("expected dist/pkg/client.js: %v", err)
	}
	// User package must not replace .forst/client contents with only pkg output.
	if _, err := os.Stat(filepath.Join(outDir, "dist", "transport.js")); err != nil {
		t.Fatalf(".forst/client must still contain transport: %v", err)
	}
	if _, err := exec.LookPath("node"); err != nil {
		return
	}
	script := filepath.Join(dir, "a07c-client.mjs")
	if err := os.WriteFile(script, []byte(`
import { Ping } from "@forst/gen/client";
if (typeof Ping !== "function") process.exit(1);
console.log("ok");
`), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runNodeRequireSmoke(t, dir, script)
	if err != nil {
		t.Fatalf("@forst/gen/client import failed: %v\n%s", err, out)
	}
}

func TestGenerate_defaultAddsNoNewTopLevelDirectory(t *testing.T) {
	dir := t.TempDir()
	ensureNodeModulesDir(t, dir)
	writeMainFt(t, dir, generateTestMinimalValidForst)

	before := map[string]struct{}{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		before[e.Name()] = struct{}{}
	}

	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}

	after, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	allowedNew := map[string]struct{}{
		".forst":       {},
		"node_modules": {},
	}
	for _, e := range after {
		name := e.Name()
		if _, ok := before[name]; ok {
			continue
		}
		if _, ok := allowedNew[name]; !ok {
			t.Fatalf("unexpected new top-level entry %q", name)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "generated")); !os.IsNotExist(err) {
		t.Fatalf("generated/ must not be created, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "client")); !os.IsNotExist(err) {
		t.Fatalf("client/ must not be created at project root, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".forst", "client")); err != nil {
		t.Fatalf("expected .forst/client: %v", err)
	}
	// Repo .gitignore covers .forst/ (module is forst/, package is cmd/forst).
	ignore, err := os.ReadFile(filepath.Join("..", "..", "..", ".gitignore"))
	if err != nil {
		t.Logf("could not read repo .gitignore for .forst coverage check: %v", err)
	} else if !strings.Contains(string(ignore), ".forst/") && !strings.Contains(string(ignore), ".forst") {
		t.Fatal("repo .gitignore must cover .forst/")
	}
}

func TestGenerate_legacyDirectoriesUntouched(t *testing.T) {
	dir := t.TempDir()
	legacyGen := filepath.Join(dir, "generated")
	legacyClient := filepath.Join(dir, "client")
	if err := os.MkdirAll(legacyGen, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(legacyClient, 0o755); err != nil {
		t.Fatal(err)
	}
	bcryptClient := []byte("// forst-owned bcrypt.client.ts marker\nexport const x = 1;\n")
	if err := os.WriteFile(filepath.Join(legacyGen, "bcrypt.client.ts"), bcryptClient, 0644); err != nil {
		t.Fatal(err)
	}
	legacyPkg := []byte(`{"name":"@forst/generated-client","version":"0.0.0"}`)
	if err := os.WriteFile(filepath.Join(legacyClient, "package.json"), legacyPkg, 0644); err != nil {
		t.Fatal(err)
	}
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	gotClient, err := os.ReadFile(filepath.Join(legacyGen, "bcrypt.client.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotClient, bcryptClient) {
		t.Fatalf("generated/bcrypt.client.ts changed")
	}
	gotPkg, err := os.ReadFile(filepath.Join(legacyClient, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotPkg, legacyPkg) {
		t.Fatalf("client/package.json changed")
	}

	// Hand-written unrelated generated/ also survives.
	hand := t.TempDir()
	if err := os.MkdirAll(filepath.Join(hand, "generated"), 0o755); err != nil {
		t.Fatal(err)
	}
	marker := []byte("not forst\n")
	if err := os.WriteFile(filepath.Join(hand, "generated", "notes.txt"), marker, 0644); err != nil {
		t.Fatal(err)
	}
	writeMainFt(t, hand, generateTestMinimalValidForst)
	if err := generateCommand([]string{hand}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(hand, "generated", "notes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, marker) {
		t.Fatal("hand-written generated/ was modified")
	}

	t.Log("alias: write ownership also covered by TestGenerate_neverWritesLegacyDirectories and TestGenerate_writesOnlyInsideOutDir")
}

func TestGenerate_editFtSeeTypes(t *testing.T) {
	t.Log("alias: byte stability and skip-writes covered by TestGenerate_isByteStableAcrossRuns and TestGenerate_secondRunRewritesNothingWhenSourcesUnchanged")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"generate":{"link":"never"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	ftPath := writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("first generate: %v", err)
	}
	first := readDistTree(t, defaultClientOutDir(dir))

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
	second := readDistTree(t, defaultClientOutDir(dir))
	if len(first) != len(second) {
		t.Fatalf("file count changed: %d -> %d", len(first), len(second))
	}
	for path, a := range first {
		if second[path] != a {
			t.Fatalf("bytes changed for %s", path)
		}
	}

	// Edit .ft to add a field; regenerate; assert .d.ts updates.
	edited := `package main

type EchoRequest = {
	message: String
	extra: String
}

func Echo(input EchoRequest) {
	return {
		echo: input.message,
		timestamp: 1234567890,
	}
}
`
	if err := os.WriteFile(ftPath, []byte(edited), 0644); err != nil {
		t.Fatal(err)
	}
	generateIO.WriteFile = origWrite
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generate after edit: %v", err)
	}
	types, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "types.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(types), "extra") {
		t.Fatalf("types.d.ts missing new field after edit:\n%s", types)
	}
}

func TestGenerate_effectCompatibility(t *testing.T) {
	t.Run("promise_mode_generated_package_has_no_effect_import", func(t *testing.T) {
		dir := t.TempDir()
		generatePromiseProject(t, dir)
		err := filepath.WalkDir(defaultClientOutDir(dir), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if !strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, ".d.ts") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if strings.Contains(string(data), `from "effect"`) || strings.Contains(string(data), "from 'effect'") {
				t.Fatalf("promise mode must not import effect: %s", path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("promise_mode_catchTag_compiles", func(t *testing.T) {
		if os.Getenv("FORST_SKIP_TS_E2E") == "1" {
			t.Skip("FORST_SKIP_TS_E2E=1")
		}
		repoEffect := findRepoEffectModule()
		if repoEffect == "" {
			t.Skip("repo effect package not found (bun install at monorepo root)")
		}
		dir := t.TempDir()
		ensureNodeModulesDir(t, dir)
		writeMainFt(t, dir, generateTestMinimalValidForst)
		if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"generate":{"link":"always"}}`), 0644); err != nil {
			t.Fatal(err)
		}
		if err := generateCommand([]string{dir}); err != nil {
			t.Fatalf("generateCommand: %v", err)
		}
		if err := os.Symlink(repoEffect, filepath.Join(dir, "node_modules", "effect")); err != nil {
			t.Fatalf("symlink effect: %v", err)
		}
		consumer, err := os.ReadFile(filepath.Join("testdata", "effect", "promise-catchtag.ts"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "promise-catchtag.ts"), consumer, 0644); err != nil {
			t.Fatal(err)
		}
		cfg := `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "noEmit": true,
    "skipLibCheck": true,
    "types": []
  },
  "include": ["promise-catchtag.ts", ".forst/client/dist/**/*.d.ts"]
}`
		if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(cfg), 0644); err != nil {
			t.Fatal(err)
		}
		if err := runTsc(t, dir); err != nil {
			t.Fatalf("promise catchTag fixture failed: %v", err)
		}
	})

	t.Run("effect_mode_return_types_and_shared_core", func(t *testing.T) {
		promiseDir := t.TempDir()
		generatePromiseProject(t, promiseDir)
		effectDir := t.TempDir()
		generateEffectProject(t, effectDir)

		effectDTS, err := os.ReadFile(filepath.Join(defaultClientDistDir(effectDir), "pkg", "main.d.ts"))
		if err != nil {
			t.Fatal(err)
		}
		got := string(effectDTS)
		for _, frag := range []string{"Effect.Effect<", "InvokeFailure", "Main", "export declare const Echo:"} {
			if !strings.Contains(got, frag) {
				t.Fatalf("missing %q in effect pkg d.ts:\n%s", frag, got)
			}
		}

		for _, rel := range []string{"core/main.js", "core/main.d.ts", "types.d.ts"} {
			p := mustRead(t, filepath.Join(defaultClientDistDir(promiseDir), rel))
			e := mustRead(t, filepath.Join(defaultClientDistDir(effectDir), rel))
			if p != e {
				t.Fatalf("%s must be byte-identical across runtimes", rel)
			}
		}

		for _, rel := range []string{"errors.js"} {
			promise := mustRead(t, filepath.Join(defaultClientDistDir(promiseDir), rel))
			if !strings.Contains(promise, `@forst/errors"`) {
				t.Fatalf("promise %s must re-export from @forst/errors", rel)
			}
			effect := mustRead(t, filepath.Join(defaultClientDistDir(effectDir), rel))
			if !strings.Contains(effect, `@forst/errors/effect"`) {
				t.Fatalf("effect %s must re-export from @forst/errors/effect", rel)
			}
		}

		promiseTree := readDistTree(t, defaultClientOutDir(promiseDir))
		effectTree := readDistTree(t, defaultClientOutDir(effectDir))
		changed := map[string]struct{}{}
		for path, content := range promiseTree {
			if effectTree[path] != content {
				changed[path] = struct{}{}
			}
		}
		for path := range effectTree {
			if _, ok := promiseTree[path]; !ok {
				changed[path] = struct{}{}
			}
		}
		allowed := map[string]struct{}{
			"dist/pkg/main.js": {}, "dist/pkg/main.d.ts": {},
			"dist/index.js": {}, "dist/index.d.ts": {},
			"dist/testing.js": {}, "dist/testing.d.ts": {},
			"dist/effect.js": {}, "dist/effect.d.ts": {},
			"dist/transport.js": {}, "dist/transport.d.ts": {},
			"dist/errors.js":   {}, "dist/errors.d.ts": {},
			"package.json": {}, "README.md": {},
		}
		for path := range changed {
			if _, ok := allowed[path]; !ok {
				t.Fatalf("unexpected differing path %q", path)
			}
		}
	})

	t.Run("effect_mode_tsc_fixture_and_runtime", func(t *testing.T) {
		if os.Getenv("FORST_SKIP_EFFECT_E2E") == "1" {
			t.Skip("FORST_SKIP_EFFECT_E2E=1")
		}
		if os.Getenv("FORST_SKIP_TS_E2E") == "1" {
			t.Skip("FORST_SKIP_TS_E2E=1")
		}
		repoEffect := findRepoEffectModule()
		if repoEffect == "" {
			t.Skip("repo effect package not found (bun install at monorepo root)")
		}

		dir := t.TempDir()
		ensureNodeModulesDir(t, dir)
		writeMainFt(t, dir, generateTestMinimalValidForst)
		if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"generate":{"effect":true,"link":"always"}}`), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(repoEffect, filepath.Join(dir, "node_modules", "effect")); err != nil {
			t.Fatalf("symlink effect: %v", err)
		}
		if err := generateCommand([]string{dir}); err != nil {
			t.Fatalf("generateCommand: %v", err)
		}

		consumer, err := os.ReadFile(filepath.Join("testdata", "effect", "consumer.ts"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "consumer.ts"), consumer, 0644); err != nil {
			t.Fatal(err)
		}
		cfg := `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "noEmit": true,
    "skipLibCheck": true,
    "types": []
  },
  "include": ["consumer.ts", ".forst/client/dist/**/*.d.ts"]
}`
		if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(cfg), 0644); err != nil {
			t.Fatal(err)
		}
		if err := runTsc(t, dir); err != nil {
			t.Fatalf("effect tsc fixture failed: %v", err)
		}

		runtime, err := os.ReadFile(filepath.Join("testdata", "effect", "runtime.mjs"))
		if err != nil {
			t.Fatal(err)
		}
		runtimePath := filepath.Join(dir, "runtime.mjs")
		if err := os.WriteFile(runtimePath, runtime, 0644); err != nil {
			t.Fatal(err)
		}
		out, err := runNodeRequireSmoke(t, dir, runtimePath)
		if err != nil {
			t.Fatalf("effect runtime fixture failed: %v\n%s", err, out)
		}
		if !strings.Contains(out, "effect-runtime-ok") {
			t.Fatalf("unexpected runtime output:\n%s", out)
		}
	})

	t.Run("effect_mode_fails_when_resolved_version_below_floor", func(t *testing.T) {
		dir := t.TempDir()
		writeMainFt(t, dir, generateTestMinimalValidForst)
		writeEffectConfig(t, dir)
		installEffectFixture(t, dir, "3.14.2")
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"effect":"^3.12.0"}}`), 0644); err != nil {
			t.Fatal(err)
		}
		err := generateCommand([]string{dir})
		if err == nil {
			t.Fatal("expected error for effect below floor")
		}
		msg := err.Error()
		for _, frag := range []string{"effect@3.14.2", ">=3.17.0"} {
			if !strings.Contains(msg, frag) {
				t.Fatalf("missing %q in:\n%s", frag, msg)
			}
		}
	})
}
