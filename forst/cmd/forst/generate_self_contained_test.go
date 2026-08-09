package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"forst/internal/ftconfig"
	transformerts "forst/internal/transformer/ts"
	"io"

	"github.com/sirupsen/logrus"
)

func defaultClientOutDir(boundary string) string {
	return filepath.Join(boundary, ".forst", "client")
}

func clientDistIncludeGlob(projectRoot string) string {
	rel, err := filepath.Rel(projectRoot, defaultClientDistDir(projectRoot))
	if err != nil {
		return ".forst/client/dist/**/*.d.ts"
	}
	return filepath.ToSlash(rel) + "/**/*.d.ts"
}

func defaultClientDistDir(boundary string) string {
	return filepath.Join(defaultClientOutDir(boundary), "dist")
}

func TestGenerate_defaultOutDirIsDotForstClient(t *testing.T) {
	dir := t.TempDir()
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	outDir := defaultClientOutDir(dir)
	for _, rel := range []string{
		"package.json",
		"README.md",
		".forst-generated",
		"dist/index.js",
		"dist/index.d.ts",
		"dist/transport.js",
		"dist/transport.d.ts",
		"dist/types.d.ts",
		"dist/core/main.js",
		"dist/core/main.d.ts",
		"dist/pkg/main.js",
		"dist/pkg/main.d.ts",
	} {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Fatalf("expected %s under default outDir: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "generated")); !os.IsNotExist(err) {
		t.Fatalf("old generated/ must not be created, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "client")); !os.IsNotExist(err) {
		t.Fatalf("old client/ must not be created, stat err=%v", err)
	}
}

func TestGenerate_writesOnlyInsideOutDir(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "ftconfig.json")
	if err := os.WriteFile(cfgPath, []byte(`{"generate":{"link":"never"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	writeMainFt(t, dir, generateTestMinimalValidForst)

	outDir := defaultClientOutDir(dir)
	var writes []string
	origW := generateIO.WriteFile
	origM := generateIO.MkdirAll
	t.Cleanup(func() {
		generateIO.WriteFile = origW
		generateIO.MkdirAll = origM
	})
	generateIO.WriteFile = func(name string, data []byte, perm os.FileMode) error {
		writes = append(writes, name)
		return origW(name, data, perm)
	}
	generateIO.MkdirAll = func(path string, perm os.FileMode) error {
		writes = append(writes, path)
		return origM(path, perm)
	}

	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}

	outClean := filepath.Clean(outDir) + string(filepath.Separator)
	for _, p := range writes {
		clean := filepath.Clean(p)
		if clean == filepath.Clean(outDir) {
			continue
		}
		if !strings.HasPrefix(clean+string(filepath.Separator), outClean) {
			t.Fatalf("generateIO path escapes outDir %s: %s", outDir, p)
		}
	}
}

func TestGenerateClientPackage_noImportEscapesOutDir(t *testing.T) {
	dir := t.TempDir()
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	distDir := defaultClientDistDir(dir)
	importRe := regexp.MustCompile(`(?m)^\s*import(?:\s+type)?\s+[\s\S]*?\s+from\s+['"]([^'"]+)['"]`)
	err := filepath.WalkDir(distDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, ".d.ts") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fileDir := filepath.Dir(path)
		for _, m := range importRe.FindAllStringSubmatch(string(data), -1) {
			spec := m[1]
			if strings.HasPrefix(spec, "@") {
				t.Fatalf("%s imports package specifier %q (must stay relative inside outDir)", path, spec)
			}
			// node: builtins are allowed for the testing module (AsyncLocalStorage).
			if strings.HasPrefix(spec, "node:") {
				continue
			}
			if !strings.HasPrefix(spec, "./") && !strings.HasPrefix(spec, "../") {
				t.Fatalf("%s import %q is not a relative path", path, spec)
			}
			// Resolve against the importing file and require the target to stay under dist/.
			cleaned := strings.TrimSuffix(spec, ".js")
			resolved := filepath.Clean(filepath.Join(fileDir, filepath.FromSlash(cleaned)))
			distClean := filepath.Clean(distDir) + string(filepath.Separator)
			if resolved != filepath.Clean(distDir) && !strings.HasPrefix(resolved+string(filepath.Separator), distClean) {
				t.Fatalf("%s import %q resolves outside dist/: %s", path, spec, resolved)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGenerateClientPackageJSON_hasNoDependencies(t *testing.T) {
	j := generateClientPackageJSON(ftconfig.EffectiveGenerateConfig(nil, ""), []string{"main"})
	var pkg map[string]any
	if err := json.Unmarshal([]byte(j), &pkg); err != nil {
		t.Fatalf("package.json not valid JSON: %v\n%s", err, j)
	}
	if _, ok := pkg["dependencies"]; ok {
		t.Fatalf("package.json must omit dependencies, got:\n%s", j)
	}
	if _, ok := pkg["devDependencies"]; ok {
		t.Fatalf("package.json must omit devDependencies, got:\n%s", j)
	}
	if pkg["sideEffects"] != false {
		t.Fatalf("sideEffects must be false, got %#v", pkg["sideEffects"])
	}
	if pkg["type"] != "module" {
		t.Fatalf("type must be module, got %#v", pkg["type"])
	}
	if _, ok := pkg["main"]; ok {
		t.Fatalf("package.json must omit top-level main in favor of exports, got:\n%s", j)
	}
	if _, ok := pkg["types"]; ok {
		t.Fatalf("package.json must omit top-level types in favor of exports, got:\n%s", j)
	}
	exports, ok := pkg["exports"].(map[string]any)
	if !ok {
		t.Fatalf("expected exports map:\n%s", j)
	}
	root := exports["."].(map[string]any)
	if root["default"] != "./dist/index.js" {
		t.Fatalf("root default must be ./dist/index.js, got %#v", root["default"])
	}
}

func TestGenerateClientPackageJSON_usesConfigPackageName(t *testing.T) {
	cfg := ftconfig.GenerateConfig{PackageName: "@acme/api-client"}
	j := generateClientPackageJSON(cfg, nil)
	if !strings.Contains(j, `"name": "@acme/api-client"`) {
		t.Fatalf("expected configured package name, got:\n%s", j)
	}
	if strings.Contains(j, "@forst/generated-client") {
		t.Fatalf("must not use legacy package name:\n%s", j)
	}
}

func TestGenerate_doesNotWriteSSRModuleWhenUnset(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "app", "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "app", "lib", "forst.invoke.ts")); !os.IsNotExist(err) {
		t.Fatalf("SSR module must not be written when ssrModule unset, stat err=%v", err)
	}
}

func TestGenerate_writesSSRModuleWhenConfigured(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "ftconfig.json")
	if err := os.WriteFile(cfgPath, []byte(`{"generate":{"ssrModule":"app/lib/forst.invoke.ts","link":"never"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "app", "lib", "forst.invoke.ts"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if strings.Contains(text, "@forst/client") {
		t.Fatalf("SSR module must not import @forst/client:\n%s", text)
	}
	if !strings.Contains(text, "getDefaultInvokeClient") {
		t.Fatalf("SSR module missing getDefaultInvokeClient:\n%s", text)
	}
	if !strings.Contains(text, "transport.js") {
		t.Fatalf("SSR module should import inlined transport:\n%s", text)
	}
	if !strings.Contains(text, "export async function Echo") {
		t.Fatalf("SSR module missing Echo export:\n%s", text)
	}
}

func TestGenerate_printsResolvedSpecifierAndExampleImport(t *testing.T) {
	dir := t.TempDir()
	writeMainFt(t, dir, generateTestMinimalValidForst)

	var buf bytes.Buffer
	prev := generateReportWriter
	t.Cleanup(func() { generateReportWriter = prev })
	generateReportWriter = &buf

	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "generate: wrote "+ftconfig.DefaultPackageName+" -> .forst/client") {
		t.Fatalf("missing resolved specifier line, got:\n%s", out)
	}
	if !strings.Contains(out, `import { Echo } from "@forst/gen/main"`) {
		t.Fatalf("missing example import, got:\n%s", out)
	}
}

func TestGenerateClientIndex_importsPackagesFromDist(t *testing.T) {
	idx := transformerts.EmitIndexESM([]string{"catalog", "orders"}, "6321", nil)
	for _, frag := range []string{
		`from "./transport.js"`,
		"createInvokeClient",
		`import { catalog } from "./pkg/catalog.js"`,
		`import { orders } from "./pkg/orders.js"`,
		"createForstClient",
		"catalog: catalog(client)",
		"orders: orders(client)",
	} {
		if !strings.Contains(idx, frag) {
			t.Fatalf("missing %q in index:\n%s", frag, idx)
		}
	}
	dts := transformerts.EmitIndexDTS([]string{"catalog", "orders"}, nil)
	if !strings.Contains(dts, `export type * from "./types.js"`) {
		t.Fatalf("index.d.ts must re-export types:\n%s", dts)
	}
	for _, frag := range []string{"InvokeRejected", "isInvokeFailure", `from "./errors.js"`} {
		if !strings.Contains(idx, frag) {
			t.Fatalf("index.js must re-export errors (%q missing):\n%s", frag, idx)
		}
		if !strings.Contains(dts, frag) {
			t.Fatalf("index.d.ts must re-export errors (%q missing):\n%s", frag, dts)
		}
	}
	for _, banned := range []string{"../generated", "@forst/client", ".client", "export { catalog, List }", "from './catalog.js'"} {
		if strings.Contains(idx, banned) {
			t.Fatalf("index must not contain %q:\n%s", banned, idx)
		}
	}
}

func TestWriteSSRModule_usesRelativeTransportImports(t *testing.T) {
	dir := t.TempDir()
	outDir := defaultClientOutDir(dir)
	distDir := filepath.Join(outDir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(io.Discard)
	outputs := []*transformerts.TypeScriptOutput{{
		PackageName: "main",
		Functions: []transformerts.FunctionSignature{
			{Name: "ListTodos", ReturnType: "ListTodosResponse"},
		},
	}}
	if err := writeSSRModule(dir, "app/lib/forst.invoke.ts", outDir, outputs, "6321", log, nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "app", "lib", "forst.invoke.ts"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "../../.forst/client/dist/transport.js") {
		t.Fatalf("expected relative transport import, got:\n%s", text)
	}
	if strings.Contains(text, "@forst/client") || strings.Contains(text, "generated/types") {
		t.Fatalf("SSR must use outDir/dist imports only:\n%s", text)
	}
}
