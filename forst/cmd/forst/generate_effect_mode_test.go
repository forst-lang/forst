package main

import (
	"encoding/json"
	"fmt"
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
	linkErrorsPackage(t, dir)
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
	linkErrorsPackage(t, dir)
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
	for _, rel := range []string{"pkg/main.js", "pkg/main.d.ts", "$transport.js"} {
		data, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), rel))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, frag := range []string{
			"invokeOptions",
			"withTransport",
			"ForstTransport",
			"EffectInvokeCallOptions",
		} {
			if strings.Contains(text, frag) {
				t.Fatalf("promise mode %s must not contain %q", rel, frag)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(defaultClientDistDir(dir), "$effect.js")); !os.IsNotExist(err) {
		t.Fatal("promise mode must not write dist/$effect.js")
	}
	if _, err := os.Stat(filepath.Join(defaultClientDistDir(dir), "$transport.js")); err != nil {
		t.Fatalf("promise mode must write dist/$transport.js: %v", err)
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
		"$main",
		"export declare class $main",
		"readonly Echo:",
	} {
		if !strings.Contains(got, frag) {
			t.Fatalf("missing %q in pkg d.ts:\n%s", frag, got)
		}
	}
	echoIdx := strings.Index(got, "readonly Echo:")
	if echoIdx < 0 {
		t.Fatal("missing Echo service member")
	}
	snippet := got[echoIdx:]
	if !strings.Contains(snippet, "Effect.Effect<") {
		t.Fatalf("Echo must return Effect:\n%s", snippet)
	}
}

func TestGenerate_effectMode_errorChannelIncludesTransportFailures(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	dts, err := os.ReadFile(filepath.Join(dist, "pkg", "main.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(dts)
	echoIdx := strings.Index(got, "readonly Echo:")
	if echoIdx < 0 {
		t.Fatal("missing Echo service member")
	}
	rest := got[echoIdx:]
	end := strings.Index(rest, ";\n")
	if end < 0 {
		t.Fatalf("malformed Echo declaration:\n%s", rest)
	}
	echoDecl := rest[:end+1]
	for _, frag := range []string{
		"Effect.Effect<",
		"InvokeFailure",
	} {
		if !strings.Contains(echoDecl, frag) {
			t.Fatalf("Echo error channel missing %q:\n%s", frag, echoDecl)
		}
	}
	if strings.Contains(echoDecl, "InvokeRejected") || strings.Contains(echoDecl, "InvokeHttpFailure") {
		t.Fatalf("Echo error channel must use compact InvokeFailure alias, not expanded invoke members:\n%s", echoDecl)
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
	j := generateClientPackageJSON(cfg, []string{"main"}, nil)
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
	if peers["@forst/cli"] != transformerts.CliPeerDependencyRange {
		t.Fatalf("@forst/cli peer = %#v, want %s", peers["@forst/cli"], transformerts.CliPeerDependencyRange)
	}
}

func TestGenerate_effectMode_packageJSONHasErrorsDependency(t *testing.T) {
	cfg := ftconfig.EffectiveGenerateConfig(nil, "")
	cfg.Effect = true
	j := generateClientPackageJSON(cfg, []string{"main"}, nil)
	var pkg map[string]any
	if err := json.Unmarshal([]byte(j), &pkg); err != nil {
		t.Fatal(err)
	}
	deps, ok := pkg["dependencies"].(map[string]any)
	if !ok {
		t.Fatalf("must have dependencies:\n%s", j)
	}
	if deps[transformerts.ErrorsPackageName] != transformerts.ErrorsDependencyRange {
		t.Fatalf("dependencies[%s] = %#v, want %s", transformerts.ErrorsPackageName, deps[transformerts.ErrorsPackageName], transformerts.ErrorsDependencyRange)
	}
}

func TestGenerate_promiseMode_packageJSONHasOptionalCliPeerOnly(t *testing.T) {
	j := generateClientPackageJSON(ftconfig.EffectiveGenerateConfig(nil, ""), []string{"main"}, nil)
	var pkg map[string]any
	if err := json.Unmarshal([]byte(j), &pkg); err != nil {
		t.Fatal(err)
	}
	peers, ok := pkg["peerDependencies"].(map[string]any)
	if !ok {
		t.Fatalf("missing peerDependencies:\n%s", j)
	}
	if peers["@forst/cli"] != transformerts.CliPeerDependencyRange {
		t.Fatalf("@forst/cli peer = %#v", peers["@forst/cli"])
	}
	if _, hasEffect := peers["effect"]; hasEffect {
		t.Fatalf("promise mode must not declare effect peer:\n%s", j)
	}
	meta, ok := pkg["peerDependenciesMeta"].(map[string]any)
	if !ok {
		t.Fatalf("missing peerDependenciesMeta:\n%s", j)
	}
	cliMeta, ok := meta["@forst/cli"].(map[string]any)
	if !ok || cliMeta["optional"] != true {
		t.Fatalf("@forst/cli must be optional peer:\n%s", j)
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
		"core.$main(client).Echo(input, invokeOptions(options, signal))",
	} {
		if !strings.Contains(got, frag) {
			t.Fatalf("missing %q:\n%s", frag, got)
		}
	}
	if strings.Contains(got, "withTransport") {
		t.Fatalf("must not emit withTransport:\n%s", got)
	}
}

func TestGenerate_effectMode_emitsNoHandWrittenAbortController(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	for _, rel := range []string{"pkg/main.js", "$transport.js", "index.js"} {
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

func TestGenerate_effectMode_errorsUseDataTaggedError(t *testing.T) {
	promiseDir := t.TempDir()
	generatePromiseProject(t, promiseDir)

	effectDir := t.TempDir()
	generateEffectProject(t, effectDir)

	for _, rel := range []string{"$errors.js"} {
		promise := mustRead(t, filepath.Join(defaultClientDistDir(promiseDir), rel))
		if !strings.Contains(promise, `use @forst/errors directly`) {
			t.Fatalf("promise %s must document @forst/errors import path:\n%s", rel, promise)
		}
		if strings.Contains(promise, `from "@forst/errors"`) {
			t.Fatalf("promise %s must not re-export from @forst/errors:\n%s", rel, promise)
		}
		if strings.Contains(promise, `from "effect"`) {
			t.Fatalf("promise %s must not import effect", rel)
		}

		effect := mustRead(t, filepath.Join(defaultClientDistDir(effectDir), rel))
		if !strings.Contains(effect, `use @forst/errors/effect directly`) {
			t.Fatalf("effect %s must document @forst/errors/effect import path:\n%s", rel, effect)
		}
		if strings.Contains(effect, `from "@forst/errors/effect"`) {
			t.Fatalf("effect %s must not re-export from @forst/errors/effect:\n%s", rel, effect)
		}
	}

	effectTesting := mustRead(t, filepath.Join(defaultClientDistDir(effectDir), "$testing.js"))
	if !strings.Contains(effectTesting, `@forst/errors/effect"`) {
		t.Fatal("effect testing.js must re-export harness error from @forst/errors/effect")
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
		"dist/pkg/main.js":            {},
		"dist/pkg/main.d.ts":          {},
		"dist/index.js":               {},
		"dist/index.d.ts":             {},
		"dist/$testing.js":            {},
		"dist/$testing.d.ts":          {},
		"dist/$transport.js":          {},
		"dist/$transport.d.ts":        {},
		"dist/transport/runtime.js":   {},
		"dist/transport/runtime.d.ts": {},
		"dist/transport/errors.js":    {},
		"dist/$errors.js":             {},
		"dist/$errors.d.ts":           {},
		"package.json":                {},
		"README.md":                   {},
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
	if !strings.Contains(js, "export class $main extends Effect.Service()") {
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
	if !strings.Contains(js, `"@forst/gen/main"`) {
		t.Fatalf("tag must include packageName:\n%s", js)
	}
	transportJS := mustRead(t, filepath.Join(dist, "$transport.js"))
	if !strings.Contains(transportJS, `"@forst/gen/Transport"`) {
		t.Fatalf("transport tag missing:\n%s", transportJS)
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
	if !strings.Contains(alpha, "export class $alpha") || strings.Contains(alpha, "export class $beta") {
		t.Fatalf("alpha module should only host $alpha:\n%s", alpha)
	}
	if !strings.Contains(beta, "export class $beta") || strings.Contains(beta, "export class $alpha") {
		t.Fatalf("beta module should only host $beta:\n%s", beta)
	}
}

func TestGenerate_effectMode_pkgModuleHasNoFlatFunctionExports(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	js := mustRead(t, filepath.Join(dist, "pkg", "main.js"))
	if strings.Contains(js, "export const { Echo }") {
		t.Fatalf("effect pkg must not re-export flat accessors:\n%s", js)
	}
	if strings.Contains(js, "export async function Echo") {
		t.Fatalf("effect pkg must not export promise flat functions:\n%s", js)
	}
	if strings.Contains(js, `export { $main } from "../core/main.js"`) {
		t.Fatalf("effect pkg must not re-export promise namespace:\n%s", js)
	}
}

func TestGenerate_effectMode_allowsSimilarPackageNames(t *testing.T) {
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
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("similar package names should succeed with $-prefixed services: %v", err)
	}
	dist := defaultClientDistDir(dir)
	for _, pkg := range []string{"user_auth", "userAuth"} {
		js := mustRead(t, filepath.Join(dist, "pkg", pkg+".js"))
		want := fmt.Sprintf("export class $%s extends Effect.Service()", pkg)
		if !strings.Contains(js, want) {
			t.Fatalf("missing %q in:\n%s", want, js)
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
	if !strings.Contains(js, "const transportLayer = ForstTransportLayer(config)") {
		t.Fatal("expected single transportLayer binding")
	}
	if strings.Count(js, "ForstTransportLayer(config)") != 1 {
		t.Fatalf("transport must be created once:\n%s", js)
	}
	if !strings.Contains(js, "Layer.provide(transportLayer)") {
		t.Fatal("expected Layer.provide(transportLayer)")
	}
}

func TestGenerate_effectMode_testingModuleEmitsPartialOverrides(t *testing.T) {
	dist := generateEffectProject(t, t.TempDir())
	dts := mustRead(t, filepath.Join(dist, "$testing.d.ts"))
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
		PackageName: "auth",
		Functions: []transformerts.FunctionSignature{{
			Name:       "VerifyToken",
			Parameters: []transformerts.Parameter{{Name: "input", Type: "$VerifyTokenRequest"}},
			ReturnType: "$VerifyTokenResponse",
		}},
		TypeImports: []string{"$VerifyTokenRequest", "$VerifyTokenResponse"},
	}}
	promiseDTS := transformerts.EmitTestingDTS(mods, "@forst/gen", transformerts.RuntimePromise)
	effectDTS := transformerts.EmitTestingEffectDTS(mods, "@forst/gen")
	for _, frag := range []string{
		"export interface ForstTestOverrides",
		"packages?:",
		"auth?: Partial<AuthHandlers>",
		"transport?: Partial<ForstInvokeClient>",
	} {
		if !strings.Contains(promiseDTS, frag) || !strings.Contains(effectDTS, frag) {
			t.Fatalf("both modes need %q", frag)
		}
	}
}

func TestGenerate_effectMode_testHandlerAcceptsValuePromiseOrEffect(t *testing.T) {
	dts := transformerts.EmitTestingEffectDTS([]transformerts.ModuleEmit{{
		PackageName: "auth",
		Functions: []transformerts.FunctionSignature{{
			Name:       "VerifyToken",
			Parameters: []transformerts.Parameter{{Name: "input", Type: "$VerifyTokenRequest"}},
			ReturnType: "$VerifyTokenResponse",
		}},
		TypeImports: []string{"$VerifyTokenRequest", "$VerifyTokenResponse"},
	}}, "@forst/gen")
	for _, frag := range []string{
		"| $VerifyTokenResponse",
		"| Promise<$VerifyTokenResponse>",
		"| Effect.Effect<$VerifyTokenResponse, InvokeFailure>",
	} {
		if !strings.Contains(dts, frag) {
			t.Fatalf("missing %q:\n%s", frag, dts)
		}
	}
}

func TestGenerate_effectMode_ForstTestLayerNeedsNoTransport(t *testing.T) {
	js := transformerts.EmitTestingEffectESM([]transformerts.ModuleEmit{{
		PackageName: "auth",
		Functions: []transformerts.FunctionSignature{{
			Name:       "VerifyToken",
			Parameters: []transformerts.Parameter{{Name: "input", Type: "$VerifyTokenRequest"}},
			ReturnType: "$VerifyTokenResponse",
		}},
	}}, "@forst/gen")
	if !strings.Contains(js, "export function ForstTestLayer") {
		t.Fatal("missing ForstTestLayer")
	}
	// ForstTestLayer body must stay mock-only; ForstTestServerLayer may use ForstTransportLayer.
	layerIdx := strings.Index(js, "export function ForstTestLayer")
	if layerIdx < 0 {
		t.Fatal("missing ForstTestLayer")
	}
	scopedIdx := strings.Index(js, "const forstTestServerScopedLayer")
	if scopedIdx < 0 {
		t.Fatal("missing forstTestServerScopedLayer helper")
	}
	layerBody := js[layerIdx:scopedIdx]
	if strings.Contains(layerBody, "ForstTransport") || strings.Contains(layerBody, "FORST_BASE_URL") || strings.Contains(layerBody, "ForstTransportLayer") {
		t.Fatalf("ForstTestLayer must not require transport:\n%s", layerBody)
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

func writeTwoPackageDomainErrorsProject(t *testing.T, dir string) {
	t.Helper()
	for _, spec := range []struct {
		pkg   string
		field string
	}{
		{pkg: "alpha", field: "message"},
		{pkg: "beta", field: "code"},
	} {
		pkgDir := filepath.Join(dir, spec.pkg)
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		src := "package " + spec.pkg + "\n\nerror NotFound {\n\t" + spec.field + ": String\n}\n\nfunc Fail() {\n\treturn NotFound{" + spec.field + `: "x"}` + "\n}\n"
		if err := os.WriteFile(filepath.Join(pkgDir, spec.pkg+".ft"), []byte(src), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGenerate_domainErrorsArePackageScoped(t *testing.T) {
	dir := t.TempDir()
	linkErrorsPackage(t, dir)
	writeTwoPackageDomainErrorsProject(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"generate":{"link":"never"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	dist := defaultClientDistDir(dir)
	errorsJS := mustRead(t, filepath.Join(dist, "transport", "errors.js"))
	for _, frag := range []string{
		`"alpha/NotFound": $alpha.$NotFound`,
		`"beta/NotFound": $beta.$NotFound`,
		`import * as $alpha from "../pkg/alpha.errors.js"`,
		`import * as $beta from "../pkg/beta.errors.js"`,
		"export function decodeDomainError",
		"packageDomainErrorRegistries",
	} {
		if !strings.Contains(errorsJS, frag) {
			t.Fatalf("missing %q in transport/errors.js:\n%s", frag, errorsJS)
		}
	}
	aggregatorJS := mustRead(t, filepath.Join(dist, "$errors.js"))
	if !strings.Contains(aggregatorJS, `export * as alpha from "./pkg/alpha.errors.js"`) {
		t.Fatalf("missing alpha namespace in $errors.js:\n%s", aggregatorJS)
	}
	errorsJS = aggregatorJS
	for _, banned := range []string{
		"DOMAIN_ERROR_REGISTRY",
		"decodeDomainError",
	} {
		if strings.Contains(errorsJS, banned) {
			t.Fatalf("errors.js must not contain %q:\n%s", banned, errorsJS)
		}
	}
	alphaErrors := mustRead(t, filepath.Join(dist, "pkg", "alpha.errors.js"))
	if !strings.Contains(alphaErrors, `extends tagged("@forst/gen/alpha/NotFound")`) {
		t.Fatalf("alpha package error tag missing:\n%s", alphaErrors)
	}
	pkgJSON, err := os.ReadFile(filepath.Join(dir, ".forst", "client", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, subpath := range []string{`"./alpha/errors"`, `"./beta/errors"`} {
		if !strings.Contains(string(pkgJSON), subpath) {
			t.Fatalf("package.json missing export %s:\n%s", subpath, pkgJSON)
		}
	}
}

func TestGenerate_httpPackageNameDoesNotClashWithRuntimeImports(t *testing.T) {
	dir := t.TempDir()
	linkErrorsPackage(t, dir)
	writePkgFT(t, dir, "http", `package http

error ServiceUnavailable {
	message: String
}

func Fail() {
	return ServiceUnavailable{message: "down"}
}
`)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	dist := defaultClientDistDir(dir)
	errorsJS := mustRead(t, filepath.Join(dist, "transport", "errors.js"))
	for _, frag := range []string{
		`import * as $http from "../pkg/http.errors.js"`,
		`"http/ServiceUnavailable": $http.$ServiceUnavailable`,
	} {
		if !strings.Contains(errorsJS, frag) {
			t.Fatalf("missing %q in transport/errors.js:\n%s", frag, errorsJS)
		}
	}
	runtimeJS := mustRead(t, filepath.Join(dist, "transport", "runtime.js"))
	if strings.Contains(runtimeJS, `import * as $http`) {
		t.Fatalf("runtime must not import domain error stars:\n%s", runtimeJS)
	}
	if !strings.Contains(runtimeJS, `import * as http from "node:http"`) {
		t.Fatalf("runtime should still import node:http:\n%s", runtimeJS)
	}
}

func TestGenerate_domainErrorsTypecheck(t *testing.T) {
	if os.Getenv("FORST_SKIP_TS_E2E") == "1" {
		t.Skip("FORST_SKIP_TS_E2E=1")
	}
	dir := t.TempDir()
	ensureNodeModulesDir(t, dir)
	linkErrorsPackage(t, dir)
	writeTwoPackageDomainErrorsProject(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	link := filepath.Join(dir, "node_modules", "@forst", "gen")
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("expected node_modules/@forst/gen link: %v", err)
	}
	assertTypeScriptCompilesSmoke(t, dir, `import { $NotFound as AlphaNotFound } from "@forst/gen/alpha/errors";
import { $NotFound as BetaNotFound } from "@forst/gen/beta/errors";
import * as alpha from "@forst/gen/alpha/errors";
import * as beta from "@forst/gen/beta/errors";
import * as errors from "@forst/gen/$errors";

const a: typeof AlphaNotFound = alpha.$NotFound;
const b: typeof BetaNotFound = beta.$NotFound;
const nsA: typeof errors.alpha.$NotFound = errors.alpha.$NotFound;
const nsB: typeof errors.beta.$NotFound = errors.beta.$NotFound;
`)
}

func keysOf(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
