package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ftconfig"
	transformerts "forst/internal/transformer/ts"
)

func TestGenerateClientPackageJSON_hasExportsMap(t *testing.T) {
	j := generateClientPackageJSON(ftconfig.EffectiveGenerateConfig(nil, ""), []string{"bcrypt", "auth"}, nil)
	var pkg map[string]any
	if err := json.Unmarshal([]byte(j), &pkg); err != nil {
		t.Fatalf("package.json not valid JSON: %v\n%s", err, j)
	}
	exports, ok := pkg["exports"].(map[string]any)
	if !ok || len(exports) == 0 {
		t.Fatalf("expected exports map, got:\n%s", j)
	}
	if _, ok := exports["."]; !ok {
		t.Fatalf("exports missing root \".\":\n%s", j)
	}
	if _, ok := exports["./bcrypt"]; !ok {
		t.Fatalf("exports missing ./bcrypt:\n%s", j)
	}
	if _, ok := exports["./auth"]; !ok {
		t.Fatalf("exports missing ./auth:\n%s", j)
	}
}

func TestGenerateClientPackageJSON_exportEntriesHaveTypesAndDefaultOnly(t *testing.T) {
	j := generateClientPackageJSON(ftconfig.EffectiveGenerateConfig(nil, ""), []string{"bcrypt"}, nil)
	var pkg map[string]any
	if err := json.Unmarshal([]byte(j), &pkg); err != nil {
		t.Fatal(err)
	}
	exports := pkg["exports"].(map[string]any)
	for key, raw := range exports {
		entry, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("export %q not an object: %#v", key, raw)
		}
		if len(entry) != 2 {
			t.Fatalf("export %q must have exactly types+default, got %d keys: %#v", key, len(entry), entry)
		}
		if _, ok := entry["types"]; !ok {
			t.Fatalf("export %q missing types", key)
		}
		if _, ok := entry["default"]; !ok {
			t.Fatalf("export %q missing default", key)
		}
		if _, ok := entry["import"]; ok {
			t.Fatalf("export %q must not have import condition", key)
		}
		if _, ok := entry["require"]; ok {
			t.Fatalf("export %q must not have require condition", key)
		}
	}
}

func TestGenerateClientPackageJSON_hasNoTypesVersionsMap(t *testing.T) {
	j := generateClientPackageJSON(ftconfig.EffectiveGenerateConfig(nil, ""), []string{"bcrypt"}, nil)
	if strings.Contains(j, "typesVersions") {
		t.Fatalf("package.json must omit typesVersions:\n%s", j)
	}
}

func TestGenerateClientPackageJSON_sideEffectsFalse(t *testing.T) {
	j := generateClientPackageJSON(ftconfig.EffectiveGenerateConfig(nil, ""), []string{"main"}, nil)
	var pkg map[string]any
	if err := json.Unmarshal([]byte(j), &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg["sideEffects"] != false {
		t.Fatalf("sideEffects must be false, got %#v", pkg["sideEffects"])
	}
	if pkg["type"] != "module" {
		t.Fatalf("type must be module, got %#v", pkg["type"])
	}
}

func TestGenerateClientPackageJSON_hasNoPublicTypesSubpath(t *testing.T) {
	j := generateClientPackageJSON(ftconfig.EffectiveGenerateConfig(nil, ""), []string{"types", "main"}, nil)
	var pkg map[string]any
	if err := json.Unmarshal([]byte(j), &pkg); err != nil {
		t.Fatal(err)
	}
	exports := pkg["exports"].(map[string]any)
	// "./types" is allowed only as a Forst package subpath, never as a compiler-owned types entry.
	// Presence of "./types" for a package named types is fine; ensure there is no separate public types key
	// that points at src/types.d.ts.
	if entry, ok := exports["./types"]; ok {
		m := entry.(map[string]any)
		typesPath, _ := m["types"].(string)
		if strings.HasSuffix(typesPath, "/types.d.ts") && !strings.Contains(typesPath, "/pkg/") {
			t.Fatalf("./types must resolve to pkg module, not compiler types file: %#v", entry)
		}
		if !strings.Contains(typesPath, "/pkg/types.d.ts") {
			t.Fatalf("./types should point at dist/pkg/types.d.ts, got %#v", entry)
		}
	}
}

func TestGenerateClientPackageJSON_doesNotExportCoreDirectory(t *testing.T) {
	j := generateClientPackageJSON(ftconfig.EffectiveGenerateConfig(nil, ""), []string{"bcrypt", "core"}, nil)
	var pkg map[string]any
	if err := json.Unmarshal([]byte(j), &pkg); err != nil {
		t.Fatal(err)
	}
	exports := pkg["exports"].(map[string]any)
	for key, raw := range exports {
		entry := raw.(map[string]any)
		for _, cond := range []string{"types", "default"} {
			path, _ := entry[cond].(string)
			if strings.Contains(path, "/core/") {
				t.Fatalf("exports must never point into core/: %s %s=%s", key, cond, path)
			}
		}
	}
	// A Forst package named core is fine as ./core -> dist/pkg/core.js
	if entry, ok := exports["./core"]; ok {
		m := entry.(map[string]any)
		if !strings.Contains(m["default"].(string), "/pkg/core.js") {
			t.Fatalf("./core should resolve to pkg/core.ts, got %#v", entry)
		}
	}
}

func TestGenerate_subpathModuleExistsPerPackage(t *testing.T) {
	dir := t.TempDir()
	writeTwoPackageProject(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	src := defaultClientDistDir(dir)
	for _, rel := range []string{
		"pkg/alpha.js",
		"pkg/beta.js",
		"core/alpha.js",
		"core/beta.js",
	} {
		if _, err := os.Stat(filepath.Join(src, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
}

func TestGenerate_userModulesLiveUnderPkgDirectory(t *testing.T) {
	dir := t.TempDir()
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	src := defaultClientDistDir(dir)
	if _, err := os.Stat(filepath.Join(src, "pkg", "main.js")); err != nil {
		t.Fatalf("expected src/pkg/main.ts: %v", err)
	}
	if _, err := os.Stat(filepath.Join(src, "main.js")); !os.IsNotExist(err) {
		t.Fatalf("flat src/main.ts must not exist, stat err=%v", err)
	}
}

func TestGenerate_promiseModePkgModuleExportsBoundNamespace(t *testing.T) {
	dir := t.TempDir()
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "pkg", "main.js"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	for _, frag := range []string{
		`import { $main as $mainCore } from "../core/main.js"`,
		"export const $main = {",
		"Echo: Object.assign(",
	} {
		if !strings.Contains(text, frag) {
			t.Fatalf("pkg module must export bound $main namespace, missing %q:\n%s", frag, text)
		}
	}
	if strings.Contains(text, `export * from "../core/main.js"`) {
		t.Fatalf("pkg module must not re-export core wholesale:\n%s", text)
	}
}

func TestGenerate_rootIndexDoesNotFlatReExportFunctions(t *testing.T) {
	dir := t.TempDir()
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	idx, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(idx)
	if !strings.Contains(text, "createForstClient") {
		t.Fatalf("root index must export createForstClient:\n%s", text)
	}
	if strings.Contains(text, "export { main, Echo }") || strings.Contains(text, "export { Echo }") {
		t.Fatalf("root must not flat re-export functions:\n%s", text)
	}
	if strings.Contains(text, "from './main.js'") {
		t.Fatalf("root must import from ./pkg/, got:\n%s", text)
	}
}

func TestGenerate_rootReExportsShapeTypes(t *testing.T) {
	idx := transformerts.EmitIndexDTS([]string{"main"}, nil, transformerts.RuntimePromise)
	if !strings.Contains(idx, `export type * from "./types.js"`) {
		t.Fatalf("root must re-export shape types:\n%s", idx)
	}
}

func TestGenerate_typesFileHasShapesOnly(t *testing.T) {
	dir := t.TempDir()
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	types, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "types.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(types)
	if !strings.Contains(text, "export interface $EchoRequest") {
		t.Fatalf("types must include shapes:\n%s", text)
	}
	if strings.Contains(text, "export function Echo(") || strings.Contains(text, "Function signatures") {
		t.Fatalf("types must not include function signatures:\n%s", text)
	}
}

func TestGenerate_sameFunctionNameInTwoPackagesSucceeds(t *testing.T) {
	dir := t.TempDir()
	writeSameFunctionTwoPackages(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("same function name in two packages must succeed: %v", err)
	}
	src := defaultClientDistDir(dir)
	for _, rel := range []string{"core/alpha.js", "core/beta.js"} {
		data, err := os.ReadFile(filepath.Join(src, rel))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "Hash") {
			t.Fatalf("%s missing Hash:\n%s", rel, data)
		}
	}
	for _, rel := range []string{"pkg/alpha.js", "pkg/beta.js"} {
		data, err := os.ReadFile(filepath.Join(src, rel))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "export const $") {
			t.Fatalf("%s must export bound namespace:\n%s", rel, data)
		}
	}
}

func TestGenerate_packageSubpathReExportsTypes(t *testing.T) {
	dir := t.TempDir()
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	pkgMod, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "pkg", "main.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(pkgMod)
	if !strings.Contains(text, "export type {") || !strings.Contains(text, "$EchoRequest") {
		t.Fatalf("pkg declarations must re-export shape types:\n%s", text)
	}
	if !strings.Contains(text, `from "../types.js"`) {
		t.Fatalf("pkg type re-export must come from ../types.js:\n%s", text)
	}
}

func writeTwoPackageProject(t *testing.T, dir string) {
	t.Helper()
	for _, pkg := range []string{"alpha", "beta"} {
		pkgDir := filepath.Join(dir, pkg)
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		src := "package " + pkg + "\n\nfunc Ping() {\n\treturn 1\n}\n"
		if err := os.WriteFile(filepath.Join(pkgDir, pkg+".ft"), []byte(src), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func writeSameFunctionTwoPackages(t *testing.T, dir string) {
	t.Helper()
	for _, pkg := range []string{"alpha", "beta"} {
		pkgDir := filepath.Join(dir, pkg)
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		src := `package ` + pkg + `

type HashRequest = {
	password: String
}

func Hash(input HashRequest) {
	return { digest: input.password }
}
`
		if err := os.WriteFile(filepath.Join(pkgDir, pkg+".ft"), []byte(src), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
