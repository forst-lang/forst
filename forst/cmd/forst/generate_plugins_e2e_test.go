package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGenerate_pluginsE2E(t *testing.T) {
	root := examplesInRoot(t)
	srcDir := filepath.Join(root, "plugins")
	if st, err := os.Stat(srcDir); err != nil || !st.IsDir() {
		t.Fatalf("example dir %s: %v", srcDir, err)
	}

	pluginDir := buildAllPlugins(t)
	t.Setenv("PATH", pluginDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	dir := t.TempDir()
	if err := copyGenerateExampleSources(srcDir, dir); err != nil {
		t.Fatalf("copy sources: %v", err)
	}
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}

	checks := []struct {
		path        string
		mustContain []string
	}{
		{"generated/jsonschema/schema.json", []string{`"$defs"`, `"minLength"`}},
		{"generated/orpc/contract.ts", []string{"PlaceOrder", "oc"}},
		{"generated/api/registry.ts", []string{"/api/orders/:id", "dispatch"}},
		{"generated/rr/routes.ts", []string{"forstApiRoutes", "api/orders/:id"}},
		{"generated/echo/manifest.txt", []string{"catalog.Catalog"}},
	}
	for _, c := range checks {
		data, err := os.ReadFile(filepath.Join(dir, c.path))
		if err != nil {
			t.Fatalf("read %s: %v", c.path, err)
		}
		text := string(data)
		for _, frag := range c.mustContain {
			if !strings.Contains(text, frag) {
				t.Fatalf("%s must contain %q\n%s", c.path, frag, text)
			}
		}
	}
}

func TestGenerate_pluginsTypecheckGeneratedTS(t *testing.T) {
	if os.Getenv("FORST_SKIP_TS_E2E") == "1" {
		t.Skip("FORST_SKIP_TS_E2E=1")
	}
	root := examplesInRoot(t)
	srcDir := filepath.Join(root, "plugins")
	pluginDir := buildAllPlugins(t)
	t.Setenv("PATH", pluginDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	dir := t.TempDir()
	if err := copyGenerateExampleSources(srcDir, dir); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	if err := writePluginTSCStubs(dir); err != nil {
		t.Fatal(err)
	}
	if err := wirePluginTSCForstGen(dir); err != nil {
		t.Fatal(err)
	}
	smoke := `import "./generated/orpc/contract.js";
import "./generated/api/registry.js";
import "./generated/rr/routes.js";
`
	if err := os.WriteFile(filepath.Join(dir, "smoke.ts"), []byte(smoke), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runTscConfig(t, dir, filepath.Join(dir, "tsconfig.json")); err != nil {
		t.Fatal(err)
	}
}

func buildAllPlugins(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, spec := range []struct {
		name string
		pkg  string
	}{
		{"forst-gen-jsonschema", "./plugins/forst-gen-jsonschema"},
		{"forst-gen-orpc", "./plugins/forst-gen-orpc"},
		{"forst-gen-file-routes", "./plugins/forst-gen-file-routes"},
		{"forst-gen-react-router", "./plugins/forst-gen-react-router"},
		{"forst-gen-echo", "./plugins/forst-gen-echo"},
	} {
		out := filepath.Join(dir, spec.name)
		buildPluginTo(t, spec.pkg, out)
	}
	return dir
}

func buildPluginTo(t *testing.T, pkg, out string) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	cmd := exec.Command("go", "build", "-o", out, pkg)
	cmd.Dir = repoRoot
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", pkg, err, outBytes)
	}
}

func writePluginTSCStubs(root string) error {
	stubs := map[string]string{
		"node_modules/zod/package.json":               `{"name":"zod","version":"0.0.0","type":"module","exports":{".":"./index.js"}}`,
		"node_modules/zod/index.js":                   `export const z = { string: () => ({ min: () => ({}), max: () => ({}), regex: () => ({}) }), number: () => ({ int: () => ({ min: () => ({}), max: () => ({}), lt: () => ({}), gt: () => ({}) }) }), boolean: () => ({}), void: () => ({}), object: () => ({ passthrough: () => ({}) }), array: () => ({}), record: () => ({}), union: () => ({}), intersection: () => ({}), tuple: () => ({}), literal: () => ({}), lazy: () => ({}), any: () => ({}), unknown: () => ({}) };`,
		"node_modules/zod/index.d.ts":                 `export declare const z: any;`,
		"node_modules/@orpc/contract/package.json":    `{"name":"@orpc/contract","version":"0.0.0","type":"module","exports":{".":"./index.js"}}`,
		"node_modules/@orpc/contract/index.js":        `export const oc = { input: () => ({ output: () => ({ errors: () => ({ route: () => ({}) }) }) }) }; export const eventIterator = () => ({});`,
		"node_modules/@orpc/contract/index.d.ts":      `export declare const oc: any; export declare function eventIterator(): any;`,
		"node_modules/react-router/package.json":      `{"name":"react-router","version":"0.0.0","type":"module","exports":{".":"./index.js"}}`,
		"node_modules/react-router/index.js":          `export {};`,
		"node_modules/react-router/index.d.ts":        `export type ActionFunctionArgs = { params: Record<string, string>; request: Request }; export type LoaderFunctionArgs = { params: Record<string, string>; request: Request };`,
		"node_modules/@react-router/dev/package.json": `{"name":"@react-router/dev","version":"0.0.0","type":"module","exports":{"./routes":"./routes.js"}}`,
		"node_modules/@react-router/dev/routes.js":    `export const route = () => ({});`,
		"node_modules/@react-router/dev/routes.d.ts":  `export declare function route(path: string, file: string): any;`,
		"node_modules/@forst/client/package.json":     `{"name":"@forst/client","version":"0.0.0","type":"module","exports":{".":"./index.js"}}`,
		"node_modules/@forst/client/index.js":         `export const createInvokeClient = () => ({ invokeFunction: async () => ({}), invokeStream: async function* () {} });`,
		"node_modules/@forst/client/index.d.ts":       `export declare function createInvokeClient(): { invokeFunction(...args: unknown[]): Promise<unknown>; invokeStream(...args: unknown[]): AsyncIterable<unknown> };`,
		"tsconfig.json": `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true,
    "skipLibCheck": true,
    "noEmit": true
  },
  "include": [
    "smoke.ts",
    "generated/orpc/**/*.ts",
    "generated/api/**/*.ts",
    "generated/rr/**/*.ts"
  ]
}`,
	}
	for rel, content := range stubs {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

func wirePluginTSCForstGen(root string) error {
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "@forst"), 0755); err != nil {
		return err
	}
	dist := defaultClientOutDir(root)
	link := filepath.Join(root, "node_modules", "@forst", "gen")
	if err := os.RemoveAll(link); err != nil {
		return err
	}
	return os.Symlink(dist, link)
}
