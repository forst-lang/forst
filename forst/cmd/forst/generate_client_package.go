package main

import (
	"fmt"
	"forst/internal/ftconfig"
	transformerts "forst/internal/transformer/ts"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// generateClientPackage writes package.json, dist/index.{js,d.ts}, and README.md under outDir.
func generateClientPackage(
	outDir string,
	genCfg ftconfig.GenerateConfig,
	outputs []*transformerts.TypeScriptOutput,
	activePackageErrors map[string]struct{},
	invokePort string,
	log *logrus.Logger,
	stats *generateWriteStats,
) error {
	if err := generateIO.MkdirAll(filepath.Join(outDir, "dist"), 0755); err != nil {
		return fmt.Errorf("failed to create dist directory: %w", err)
	}

	packageNames := make([]string, 0, len(outputs))
	for _, out := range outputs {
		if out != nil && out.PackageName != "" {
			packageNames = append(packageNames, out.PackageName)
		}
	}
	runtime := transformerts.RuntimeFromConfig(genCfg)

	var domainParts [][]transformerts.ErrorClass
	for _, out := range outputs {
		if out != nil && len(out.DomainErrors) > 0 {
			domainParts = append(domainParts, out.DomainErrors)
		}
	}
	domainErrors, err := transformerts.MergeDomainErrors(domainParts...)
	if err != nil {
		return err
	}

	indexJS := transformerts.EmitIndexESM(packageNames, invokePort, domainErrors, runtime)
	if runtime == transformerts.RuntimeEffect {
		indexJS += transformerts.EmitIndexEffectESM(packageNames, genCfg.PackageName)
	}
	indexJSPath := filepath.Join(outDir, "dist", "index.js")
	if err := writeGeneratedFile(indexJSPath, []byte(indexJS), stats); err != nil {
		return fmt.Errorf("failed to write client index.js: %w", err)
	}
	log.Debugf("Generated client index: %s", indexJSPath)

	indexDTS := transformerts.EmitIndexDTS(packageNames, domainErrors, runtime)
	if runtime == transformerts.RuntimeEffect {
		indexDTS += transformerts.EmitIndexEffectDTS(packageNames)
	}
	indexDTSPath := filepath.Join(outDir, "dist", "index.d.ts")
	if err := writeGeneratedFile(indexDTSPath, []byte(indexDTS), stats); err != nil {
		return fmt.Errorf("failed to write client index.d.ts: %w", err)
	}

	modules := make([]transformerts.ModuleEmit, 0, len(outputs))
	for _, out := range outputs {
		if out == nil || out.PackageName == "" {
			continue
		}
		modules = append(modules, transformerts.ModuleEmitFromOutput(out, false))
	}
	testingKey := genCfg.EffectiveTestingSubpath()
	var testingJS string
	var testingDTS string
	if runtime == transformerts.RuntimeEffect {
		testingJS = transformerts.EmitTestingEffectESM(modules, genCfg.PackageName)
		testingDTS = transformerts.EmitTestingEffectDTS(modules, genCfg.PackageName)
	} else {
		testingJS = transformerts.EmitTestingESM(modules, genCfg.PackageName, transformerts.RuntimePromise)
		testingDTS = transformerts.EmitTestingDTS(modules, genCfg.PackageName, transformerts.RuntimePromise)
	}
	testingJSPath := filepath.Join(outDir, "dist", testingKey+".js")
	if err := writeGeneratedFile(testingJSPath, []byte(testingJS), stats); err != nil {
		return fmt.Errorf("failed to write testing.js: %w", err)
	}
	log.Debugf("Generated testing module: %s", testingJSPath)

	testingDTSPath := filepath.Join(outDir, "dist", testingKey+".d.ts")
	if err := writeGeneratedFile(testingDTSPath, []byte(testingDTS), stats); err != nil {
		return fmt.Errorf("failed to write testing.d.ts: %w", err)
	}

	packageContent := generateClientPackageJSON(genCfg, packageNames, sortedMapKeys(activePackageErrors))
	packagePath := filepath.Join(outDir, "package.json")
	if err := writeGeneratedFile(packagePath, []byte(packageContent), stats); err != nil {
		return fmt.Errorf("failed to write client package.json: %w", err)
	}
	log.Debugf("Generated client package.json: %s", packagePath)

	readme := generateClientREADME(genCfg, invokePort, outputs)
	readmePath := filepath.Join(outDir, "README.md")
	if err := writeGeneratedFile(readmePath, []byte(readme), stats); err != nil {
		return fmt.Errorf("failed to write client README: %w", err)
	}
	log.Debugf("Generated client README: %s", readmePath)

	return nil
}

// generateClientPackageJSON creates package.json with an exports map pointing at dist/.
func generateClientPackageJSON(genCfg ftconfig.GenerateConfig, packages, domainErrorPackages []string) string {
	name := genCfg.PackageName
	if name == "" {
		name = ftconfig.DefaultPackageName
	}
	unique := sortedUniqueNonEmpty(packages)

	var b strings.Builder
	b.WriteString("{\n")
	fmt.Fprintf(&b, "  \"name\": %s,\n", jsonString(name))
	b.WriteString("  \"private\": true,\n")
	b.WriteString("  \"version\": \"0.0.0\",\n")
	b.WriteString("  \"description\": \"Auto-generated Forst client\",\n")
	b.WriteString("  \"type\": \"module\",\n")
	b.WriteString("  \"sideEffects\": false,\n")
	b.WriteString("  \"engines\": {\n")
	b.WriteString("    \"node\": \">=20.19\"\n")
	b.WriteString("  },\n")
	b.WriteString("  \"exports\": {\n")
	b.WriteString("    \".\": {\n")
	b.WriteString("      \"types\": \"./dist/index.d.ts\",\n")
	b.WriteString("      \"default\": \"./dist/index.js\"\n")
	b.WriteString("    }")
	for _, pkg := range unique {
		appendPackageJSONExport(&b, "./"+pkg, "./dist/pkg/"+pkg+".d.ts", "./dist/pkg/"+pkg+".js")
	}
	domainErrPkgs := sortedUniqueNonEmpty(domainErrorPackages)
	for _, pkg := range domainErrPkgs {
		stem := transformerts.PackageDomainErrorsFileStem(pkg)
		appendPackageJSONExport(&b, "./"+pkg+"/errors", "./dist/pkg/"+stem+".d.ts", "./dist/pkg/"+stem+".js")
	}
	testingKey := genCfg.EffectiveTestingSubpath()
	appendPackageJSONExport(&b, "./"+testingKey, "./dist/"+testingKey+".d.ts", "./dist/"+testingKey+".js")
	appendPackageJSONExport(
		&b,
		"./"+transformerts.InfraErrorsSubpath,
		"./dist/"+transformerts.InfraErrorsSubpath+".d.ts",
		"./dist/"+transformerts.InfraErrorsSubpath+".js",
	)
	appendPackageJSONExport(
		&b,
		"./"+transformerts.InfraTransportSubpath,
		"./dist/"+transformerts.InfraTransportSubpath+".d.ts",
		"./dist/"+transformerts.InfraTransportSubpath+".js",
	)
	b.WriteString("\n  },\n")
	b.WriteString("  \"dependencies\": {\n")
	fmt.Fprintf(&b, "    %q: %s\n", transformerts.ErrorsPackageName, jsonString(transformerts.ErrorsDependencyRange))
	b.WriteString("  },\n")
	b.WriteString("  \"peerDependencies\": {\n")
	fmt.Fprintf(&b, "    \"@forst/cli\": %s", jsonString(transformerts.CliPeerDependencyRange))
	if genCfg.Effect {
		b.WriteString(",\n")
		fmt.Fprintf(&b, "    \"effect\": %s\n", jsonString(transformerts.EffectPeerDependencyRange))
	} else {
		b.WriteString("\n")
	}
	b.WriteString("  },\n")
	b.WriteString("  \"peerDependenciesMeta\": {\n")
	b.WriteString("    \"@forst/cli\": {\n")
	b.WriteString("      \"optional\": true\n")
	b.WriteString("    }\n")
	b.WriteString("  }\n")
	b.WriteString("}\n")
	return b.String()
}

// appendPackageJSONExport writes one exports map entry (types + default).
func appendPackageJSONExport(b *strings.Builder, subpath, typesPath, defaultPath string) {
	b.WriteString(",\n")
	fmt.Fprintf(b, "    %s: {\n", jsonString(subpath))
	fmt.Fprintf(b, "      \"types\": %s,\n", jsonString(typesPath))
	fmt.Fprintf(b, "      \"default\": %s\n", jsonString(defaultPath))
	b.WriteString("    }")
}

func jsonString(s string) string {
	return strconv.Quote(s)
}

func sortedUniqueNonEmpty(in []string) []string {
	sorted := append([]string(nil), in...)
	sort.Strings(sorted)
	seen := make(map[string]struct{}, len(sorted))
	var unique []string
	for _, s := range sorted {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		unique = append(unique, s)
	}
	return unique
}

func sortedMapKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return sortedUniqueNonEmpty(keys)
}


// generateClientREADME documents the resolved specifier, invoke env, and postinstall line.
func generateClientREADME(genCfg ftconfig.GenerateConfig, invokePort string, outputs []*transformerts.TypeScriptOutput) string {
	name := genCfg.PackageName
	if name == "" {
		name = ftconfig.DefaultPackageName
	}
	if invokePort == "" {
		invokePort = ftconfig.DefaultEmbeddedInvokePort
	}

	var b strings.Builder
	b.WriteString("# Generated Forst client\n\n")
	b.WriteString("This package is produced by `forst generate`. Do not edit by hand.\n\n")
	fmt.Fprintf(&b, "## Import specifier\n\n`%s`\n\n", name)
	if example, ok := exampleImportLine(name, outputs); ok {
		b.WriteString("Example:\n\n```ts\n")
		b.WriteString(example)
		b.WriteString("\n```\n\n")
	}
	b.WriteString("## Invoke server\n\n")
	b.WriteString("Connect-only transport. Set one of:\n\n")
	b.WriteString("- `FORST_INVOKE_URL`\n")
	b.WriteString("- `FORST_BASE_URL`\n")
	b.WriteString("- `FORST_DEV_URL`\n\n")
	fmt.Fprintf(&b, "Default fallback: `http://127.0.0.1:%s`\n\n", invokePort)
	b.WriteString("## Errors\n\n")
	fmt.Fprintf(&b, "- Runtime dependency: `%s` %s (shared invoke/harness/unknown failure classes).\n", transformerts.ErrorsPackageName, transformerts.ErrorsDependencyRange)
	b.WriteString("- Shared invoke re-exports: `" + name + "/" + transformerts.InfraErrorsSubpath + "`.\n")
	b.WriteString("- Per-package domain errors: `" + name + "/<forstPackage>/errors`.\n")
	b.WriteString("- Forst package names must be Go-safe identifiers (letters, digits, underscores). Names containing `$` are rejected (`$` prefixes are reserved for compiler infra subpaths like `" + transformerts.DefaultInfraTestingSubpath + "`).\n")
	b.WriteString("- Built-in invoke/harness/unknown failure `_tag` values use the `@forst/errors/` prefix. Domain error tags use `@<npmPackage>/<forstPackage>/<ErrorName>`.\n")
	b.WriteString("- Prefer matching on `_tag` in Effect code. In Promise code, `instanceof` works for shared invoke failures.\n\n")
	b.WriteString("## Lifecycle script\n\n")
	b.WriteString("Ephemeral output under `.forst/` is gitignored. Do not commit this directory. ")
	b.WriteString("The package `version` stays at `0.0.0` because it is private, regenerated on every `forst generate`, and never published to npm.\n\n")
	b.WriteString("Add a postinstall script so fresh checkouts regenerate and relink:\n\n")
	b.WriteString("```json\n")
	b.WriteString(`{ "scripts": { "postinstall": "forst generate ." } }`)
	b.WriteString("\n```\n")
	b.WriteString("\n## Real-server tests\n\n")
	fmt.Fprintf(&b, "Optional peer: `@forst/cli` %s (install with `%s` to use `startForstTestServer` / `ForstTestServerLayer`).\n",
		transformerts.CliPeerDependencyRange, transformerts.CliInstallCommand)
	if genCfg.Effect {
		b.WriteString("\n## Effect mode\n\n")
		b.WriteString("This package was generated with `generate.effect: true`.\n\n")
		fmt.Fprintf(&b, "- Peer dependency: `effect` %s (required for `Layer.mock`).\n", transformerts.EffectPeerDependencyRange)
		b.WriteString("- Call sites return `Effect.Effect<Response, InvokeFailure, PkgService>` and need `Effect.provide(ForstClientLive)` (or `ForstClientLayer`).\n")
		b.WriteString("- Mocking: `Layer.mock($pkg, { ... })` for one service, `ForstTestLayer(overrides)` for the whole client, or `Layer.mock(ForstTransport, { client })` for the wire (`@forst/gen/$transport`).\n")
		b.WriteString("- Real server: `ForstTestServerLayer` / `makeForstTestServer` (needs optional `@forst/cli` peer).\n")
		b.WriteString("- Invoke, domain, and harness errors use `Data.TaggedError` from the effect peer. `Equal.equals` works on error values.\n")
		b.WriteString("- Prefer `Effect.retry` over `options.retries` (omitted in Effect mode).\n")
	}
	return b.String()
}
