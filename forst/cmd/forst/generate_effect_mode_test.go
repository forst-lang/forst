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

func writeEffectConfig(t *testing.T, dir string) {
	t.Helper()
	cfg := `{"generate":{"effect":true,"link":"never"}}`
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
}

func installEffectFixture(t *testing.T, dir, version string) {
	t.Helper()
	pkgDir := filepath.Join(dir, "node_modules", "effect")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"name":"effect","version":` + jsonString(version) + `}`
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func generateEffectProject(t *testing.T, dir string) string {
	t.Helper()
	writeMainFt(t, dir, generateTestMinimalValidForst)
	writeEffectConfig(t, dir)
	installEffectFixture(t, dir, "3.21.4")
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	return defaultClientDistDir(dir)
}

func generatePromiseProject(t *testing.T, dir string) string {
	t.Helper()
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"generate":{"link":"never"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	return defaultClientDistDir(dir)
}

func TestGenerate_effectDefaultsToPromiseRuntime(t *testing.T) {
	cfg := ftconfig.EffectiveGenerateConfig(nil, "")
	if transformerts.RuntimeFromConfig(cfg) != transformerts.RuntimePromise {
		t.Fatal("default runtime must be Promise")
	}
	dir := t.TempDir()
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	pkg, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "pkg", "main.js"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(pkg), "Effect.tryPromise") {
		t.Fatal("default generate must not emit Effect wrappers")
	}
	if _, err := os.Stat(filepath.Join(defaultClientDistDir(dir), "effect.js")); !os.IsNotExist(err) {
		t.Fatal("promise mode must not write dist/effect.js")
	}
}

func TestGenerate_effectMode_functionsReturnEffectType(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	dts, err := os.ReadFile(filepath.Join(dist, "pkg", "main.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(dts)
	for _, frag := range []string{
		"Effect.Effect<",
		"InvokeFailure",
		"Main",
		"export declare const Echo:",
	} {
		if !strings.Contains(got, frag) {
			t.Fatalf("missing %q in pkg d.ts:\n%s", frag, got)
		}
	}
	echoIdx := strings.Index(got, "export declare const Echo:")
	if echoIdx < 0 {
		t.Fatal("missing Echo const")
	}
	snippet := got[echoIdx:]
	if !strings.Contains(snippet, "Effect.Effect<") {
		t.Fatalf("Echo must return Effect:\n%s", snippet)
	}
}

func TestGenerate_effectMode_errorChannelIsInvokeFailure(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	dts, err := os.ReadFile(filepath.Join(dist, "pkg", "main.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dts), "Effect.Effect<") || !strings.Contains(string(dts), "InvokeFailure") {
		t.Fatalf("error channel must be InvokeFailure:\n%s", dts)
	}
}

func TestGenerate_effectMode_omitsSafeNamespace(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	dts, err := os.ReadFile(filepath.Join(dist, "pkg", "main.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dts), "namespace Echo") || strings.Contains(string(dts), ".safe") {
		t.Fatalf("Effect pkg must omit .safe:\n%s", dts)
	}
	js, err := os.ReadFile(filepath.Join(dist, "pkg", "main.js"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(js), ".safe") {
		t.Fatalf("Effect pkg js must omit .safe:\n%s", js)
	}
}

func TestGenerate_effectMode_omitsRetriesOption(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	dts, err := os.ReadFile(filepath.Join(dist, "pkg", "main.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(dts)
	if !strings.Contains(got, "EffectInvokeCallOptions") {
		t.Fatal("expected EffectInvokeCallOptions")
	}
	if !strings.Contains(got, `Omit<InvokeCallOptions, "retries">`) {
		t.Fatalf("retries must be omitted from Effect options:\n%s", got)
	}
}

func TestGenerate_effectMode_packageJSONHasEffectPeerDependency(t *testing.T) {
	cfg := ftconfig.EffectiveGenerateConfig(nil, "")
	cfg.Effect = true
	j := generateClientPackageJSON(cfg, []string{"main"})
	var pkg map[string]any
	if err := json.Unmarshal([]byte(j), &pkg); err != nil {
		t.Fatal(err)
	}
	peers, ok := pkg["peerDependencies"].(map[string]any)
	if !ok {
		t.Fatalf("missing peerDependencies:\n%s", j)
	}
	if peers["effect"] != transformerts.EffectPeerDependencyRange {
		t.Fatalf("effect peer = %#v, want %s", peers["effect"], transformerts.EffectPeerDependencyRange)
	}
}

func TestGenerate_effectMode_packageJSONHasNoRuntimeDependencies(t *testing.T) {
	cfg := ftconfig.EffectiveGenerateConfig(nil, "")
	cfg.Effect = true
	j := generateClientPackageJSON(cfg, []string{"main"})
	var pkg map[string]any
	if err := json.Unmarshal([]byte(j), &pkg); err != nil {
		t.Fatal(err)
	}
	if _, ok := pkg["dependencies"]; ok {
		t.Fatalf("must not have dependencies:\n%s", j)
	}
}

func TestGenerate_promiseMode_packageJSONHasNoPeerDependencies(t *testing.T) {
	j := generateClientPackageJSON(ftconfig.EffectiveGenerateConfig(nil, ""), []string{"main"})
	var pkg map[string]any
	if err := json.Unmarshal([]byte(j), &pkg); err != nil {
		t.Fatal(err)
	}
	if _, ok := pkg["peerDependencies"]; ok {
		t.Fatalf("promise mode must not have peerDependencies:\n%s", j)
	}
}

func TestGenerate_effectMode_failsWhenEffectNotInstalled(t *testing.T) {
	dir := t.TempDir()
	writeMainFt(t, dir, generateTestMinimalValidForst)
	writeEffectConfig(t, dir)
	err := generateCommand([]string{dir})
	if err == nil {
		t.Fatal("expected error when effect missing")
	}
	msg := err.Error()
	for _, frag := range []string{"generate.effect", "found:    none", ">=3.17.0"} {
		if !strings.Contains(msg, frag) {
			t.Fatalf("missing %q in:\n%s", frag, msg)
		}
	}
}

func TestGenerate_effectMode_failsWhenEffectResolvesBelowFloor(t *testing.T) {
	dir := t.TempDir()
	writeMainFt(t, dir, generateTestMinimalValidForst)
	writeEffectConfig(t, dir)
	installEffectFixture(t, dir, "3.14.2")
	// Declared range looks fine; resolved version must be checked.
	rootPkg := `{"dependencies":{"effect":"^3.12.0"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(rootPkg), 0644); err != nil {
		t.Fatal(err)
	}
	err := generateCommand([]string{dir})
	if err == nil {
		t.Fatal("expected error for effect below floor")
	}
	msg := err.Error()
	for _, frag := range []string{"effect@3.14.2", ">=3.17.0", "node_modules/effect"} {
		if !strings.Contains(msg, frag) {
			t.Fatalf("missing %q in:\n%s", frag, msg)
		}
	}
}

func TestGenerate_effectMode_serviceUsesTryPromiseWithSuppliedSignal(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	js, err := os.ReadFile(filepath.Join(dist, "pkg", "main.js"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(js)
	for _, frag := range []string{
		"Effect.tryPromise",
		"try: (signal) =>",
		"withTransport(client, options, signal)",
	} {
		if !strings.Contains(got, frag) {
			t.Fatalf("missing %q:\n%s", frag, got)
		}
	}
}

func TestGenerate_effectMode_emitsNoHandWrittenAbortController(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	for _, rel := range []string{"pkg/main.js", "effect.js", "index.js"} {
		data, err := os.ReadFile(filepath.Join(dist, rel))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "AbortController") {
			t.Fatalf("%s must not hand-write AbortController", rel)
		}
	}
}

func TestGenerate_effectMode_coreModuleIsByteIdenticalToPromiseMode(t *testing.T) {
	promiseDir := t.TempDir()
	generatePromiseProject(t, promiseDir)

	effectDir := t.TempDir()
	generateEffectProject(t, effectDir)

	promiseCore := mustRead(t, filepath.Join(defaultClientDistDir(promiseDir), "core", "main.js"))
	effectCore := mustRead(t, filepath.Join(defaultClientDistDir(effectDir), "core", "main.js"))
	if promiseCore != effectCore {
		t.Fatal("dist/core/main.js must be byte-identical across runtimes")
	}
	promiseCoreDTS := mustRead(t, filepath.Join(defaultClientDistDir(promiseDir), "core", "main.d.ts"))
	effectCoreDTS := mustRead(t, filepath.Join(defaultClientDistDir(effectDir), "core", "main.d.ts"))
	if promiseCoreDTS != effectCoreDTS {
		t.Fatal("dist/core/main.d.ts must be byte-identical across runtimes")
	}
}

func TestGenerate_effectMode_transportAndErrorsAreByteIdenticalToPromiseMode(t *testing.T) {
	promiseDir := t.TempDir()
	generatePromiseProject(t, promiseDir)

	effectDir := t.TempDir()
	generateEffectProject(t, effectDir)

	for _, rel := range []string{"transport.js", "transport.d.ts", "errors.js", "errors.d.ts", "types.d.ts"} {
		p := mustRead(t, filepath.Join(defaultClientDistDir(promiseDir), rel))
		e := mustRead(t, filepath.Join(defaultClientDistDir(effectDir), rel))
		if p != e {
			t.Fatalf("%s must be byte-identical across runtimes", rel)
		}
	}
}

func TestGenerate_effectMode_onlyPkgModulesDifferFromPromiseMode(t *testing.T) {
	promiseDir := t.TempDir()
	generatePromiseProject(t, promiseDir)

	effectDir := t.TempDir()
	generateEffectProject(t, effectDir)

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
		"dist/pkg/main.js":   {},
		"dist/pkg/main.d.ts": {},
		"dist/index.js":      {},
		"dist/index.d.ts":    {},
		"dist/testing.js":    {},
		"dist/testing.d.ts":  {},
		"dist/effect.js":     {},
		"dist/effect.d.ts":   {},
		"package.json":       {},
		"README.md":          {},
	}
	for path := range changed {
		if _, ok := allowed[path]; !ok {
			t.Fatalf("unexpected differing path %q (changed=%v)", path, keysOf(changed))
		}
	}
	for _, must := range []string{"dist/pkg/main.js", "dist/pkg/main.d.ts"} {
		if _, ok := changed[must]; !ok {
			t.Fatalf("expected %s to differ", must)
		}
	}
}

func TestGenerate_promiseMode_emitsNoEffectImport(t *testing.T) {
	dir := t.TempDir()
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
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
}

func TestGenerate_effectMode_coreModulesContainNoEffectImport(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	core := mustRead(t, filepath.Join(dist, "core", "main.js"))
	if strings.Contains(core, `from "effect"`) || strings.Contains(core, "from 'effect'") {
		t.Fatalf("core must not import effect:\n%s", core)
	}
}

func TestGenerate_effectMode_emitsServiceClassPerPackage(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	js := mustRead(t, filepath.Join(dist, "pkg", "main.js"))
	if !strings.Contains(js, "export class Main extends Effect.Service()") {
		t.Fatalf("missing service class:\n%s", js)
	}
}

func TestGenerate_effectMode_serviceDeclaresAccessorsTrue(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	js := mustRead(t, filepath.Join(dist, "pkg", "main.js"))
	if !strings.Contains(js, "accessors: true") {
		t.Fatal("missing accessors: true")
	}
}

func TestGenerate_effectMode_serviceDeclaresTransportDependency(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	js := mustRead(t, filepath.Join(dist, "pkg", "main.js"))
	if !strings.Contains(js, "dependencies: [ForstTransport.Default]") {
		t.Fatal("missing ForstTransport dependency")
	}
}

func TestGenerate_effectMode_tagStringIncludesPackageName(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	js := mustRead(t, filepath.Join(dist, "pkg", "main.js"))
	if !strings.Contains(js, `"@forst/gen/Main"`) {
		t.Fatalf("tag must include packageName:\n%s", js)
	}
	effectJS := mustRead(t, filepath.Join(dist, "effect.js"))
	if !strings.Contains(effectJS, `"@forst/gen/Transport"`) {
		t.Fatalf("transport tag missing:\n%s", effectJS)
	}
}

func TestGenerate_effectMode_serviceLivesInItsOwnSubpathModule(t *testing.T) {
	dir := t.TempDir()
	writeTwoPackageProject(t, dir)
	writeEffectConfig(t, dir)
	installEffectFixture(t, dir, "3.21.4")
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	dist := defaultClientDistDir(dir)
	alpha := mustRead(t, filepath.Join(dist, "pkg", "alpha.js"))
	beta := mustRead(t, filepath.Join(dist, "pkg", "beta.js"))
	if !strings.Contains(alpha, "export class Alpha") || strings.Contains(alpha, "export class Beta") {
		t.Fatalf("alpha module should only host Alpha:\n%s", alpha)
	}
	if !strings.Contains(beta, "export class Beta") || strings.Contains(beta, "export class Alpha") {
		t.Fatalf("beta module should only host Beta:\n%s", beta)
	}
}

func TestGenerate_effectMode_subpathReExportsServiceAccessors(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	js := mustRead(t, filepath.Join(dist, "pkg", "main.js"))
	if !strings.Contains(js, "export const { Echo } = Main") {
		t.Fatalf("missing accessor re-export:\n%s", js)
	}
}

func TestGenerate_effectMode_failsOnServiceClassNameCollision(t *testing.T) {
	dir := t.TempDir()
	for _, pkg := range []string{"user_auth", "userAuth"} {
		pkgDir := filepath.Join(dir, pkg)
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		src := "package " + pkg + "\n\nfunc Ping() {\n\treturn 1\n}\n"
		if err := os.WriteFile(filepath.Join(pkgDir, pkg+".ft"), []byte(src), 0644); err != nil {
			t.Fatal(err)
		}
	}
	writeEffectConfig(t, dir)
	installEffectFixture(t, dir, "3.21.4")
	err := generateCommand([]string{dir})
	if err == nil {
		t.Fatal("expected service class collision error")
	}
	msg := err.Error()
	for _, frag := range []string{"user_auth", "userAuth", "UserAuth"} {
		if !strings.Contains(msg, frag) {
			t.Fatalf("missing %q in:\n%s", frag, msg)
		}
	}
}

func TestGenerate_effectMode_rootEmitsForstClientLiveAndRuntime(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	js := mustRead(t, filepath.Join(dist, "index.js"))
	for _, frag := range []string{
		"export const ForstClientLive",
		"export const ForstClientLayer",
		"export const makeForstClientRuntime",
		"ManagedRuntime.make",
	} {
		if !strings.Contains(js, frag) {
			t.Fatalf("missing %q in index.js:\n%s", frag, js)
		}
	}
	dts := mustRead(t, filepath.Join(dist, "index.d.ts"))
	for _, frag := range []string{"ForstClientLive", "ForstClientLayer", "makeForstClientRuntime"} {
		if !strings.Contains(dts, frag) {
			t.Fatalf("missing %q in index.d.ts", frag)
		}
	}
}

func TestGenerate_effectMode_layerSharesOneTransportAcrossPackages(t *testing.T) {
	dir := t.TempDir()
	writeTwoPackageProject(t, dir)
	writeEffectConfig(t, dir)
	installEffectFixture(t, dir, "3.21.4")
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	js := mustRead(t, filepath.Join(defaultClientDistDir(dir), "index.js"))
	if !strings.Contains(js, "const transportLayer = layerTransport(config)") {
		t.Fatal("expected single transportLayer binding")
	}
	if strings.Count(js, "layerTransport(config)") != 1 {
		t.Fatalf("transport must be created once:\n%s", js)
	}
	if !strings.Contains(js, "Layer.provide(transportLayer)") {
		t.Fatal("expected Layer.provide(transportLayer)")
	}
}

func TestGenerate_effectMode_testingModuleEmitsPartialOverrides(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	dts := mustRead(t, filepath.Join(dist, "testing.d.ts"))
	for _, frag := range []string{
		"export interface ForstTestOverrides",
		"packages?:",
		"main?: Partial<MainHandlers>",
		"transport?: Partial<ForstInvokeClient>",
		"ForstTestLayer",
	} {
		if !strings.Contains(dts, frag) {
			t.Fatalf("missing %q:\n%s", frag, dts)
		}
	}
}

func TestGenerate_effectMode_overridesShapeMatchesPromiseMode(t *testing.T) {
	mods := []transformerts.ModuleEmit{{
		PackageName: "bcrypt",
		Functions: []transformerts.FunctionSignature{{
			Name:       "ComparePassword",
			Parameters: []transformerts.Parameter{{Name: "input", Type: "ComparePasswordRequest"}},
			ReturnType: "ComparePasswordResponse",
		}},
		TypeImports: []string{"ComparePasswordRequest", "ComparePasswordResponse"},
	}}
	promiseDTS := transformerts.EmitTestingDTS(mods)
	effectDTS := transformerts.EmitTestingEffectDTS(mods)
	for _, frag := range []string{
		"export interface ForstTestOverrides",
		"packages?:",
		"bcrypt?: Partial<BcryptHandlers>",
		"transport?: Partial<ForstInvokeClient>",
	} {
		if !strings.Contains(promiseDTS, frag) || !strings.Contains(effectDTS, frag) {
			t.Fatalf("both modes need %q", frag)
		}
	}
}

func TestGenerate_effectMode_testHandlerAcceptsValuePromiseOrEffect(t *testing.T) {
	dts := transformerts.EmitTestingEffectDTS([]transformerts.ModuleEmit{{
		PackageName: "bcrypt",
		Functions: []transformerts.FunctionSignature{{
			Name:       "ComparePassword",
			Parameters: []transformerts.Parameter{{Name: "input", Type: "ComparePasswordRequest"}},
			ReturnType: "ComparePasswordResponse",
		}},
		TypeImports: []string{"ComparePasswordRequest", "ComparePasswordResponse"},
	}})
	for _, frag := range []string{
		"| ComparePasswordResponse",
		"| Promise<ComparePasswordResponse>",
		"| Effect.Effect<ComparePasswordResponse, InvokeFailure>",
	} {
		if !strings.Contains(dts, frag) {
			t.Fatalf("missing %q:\n%s", frag, dts)
		}
	}
}

func TestGenerate_effectMode_ForstTestLayerNeedsNoTransport(t *testing.T) {
	js := transformerts.EmitTestingEffectESM([]transformerts.ModuleEmit{{
		PackageName: "bcrypt",
		Functions: []transformerts.FunctionSignature{{
			Name:       "ComparePassword",
			Parameters: []transformerts.Parameter{{Name: "input", Type: "ComparePasswordRequest"}},
			ReturnType: "ComparePasswordResponse",
		}},
	}})
	if !strings.Contains(js, "export function ForstTestLayer") {
		t.Fatal("missing ForstTestLayer")
	}
	if strings.Contains(js, "ForstTransport") || strings.Contains(js, "FORST_BASE_URL") {
		t.Fatalf("ForstTestLayer must not require transport:\n%s", js)
	}
	if !strings.Contains(js, "Layer.mock") {
		t.Fatal("must delegate to Layer.mock")
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func keysOf(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
