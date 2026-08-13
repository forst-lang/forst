package main

import (
	"fmt"
	"forst/internal/ftconfig"
	transformerts "forst/internal/transformer/ts"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

// reservedDistFiles are never pruned as stale package modules under outDir/dist/.
var reservedDistFiles = map[string]struct{}{
	"index.js":        {},
	"index.d.ts":      {},
	"transport.js":    {},
	"transport.d.ts":  {},
	"types.d.ts":      {},
	"$errors.js":      {},
	"$errors.d.ts":    {},
	"$effect.js":      {},
	"$effect.d.ts":    {},
	"$testing.js":     {},
	"$testing.d.ts":   {},
}

// reservedDistFileSet returns compiler-owned dist root files, including the configured testing subpath.
func reservedDistFileSet(testingSubpath string) map[string]struct{} {
	out := make(map[string]struct{}, len(reservedDistFiles)+2)
	for k, v := range reservedDistFiles {
		out[k] = v
	}
	key := testingSubpath
	if key == "" {
		key = transformerts.DefaultInfraTestingSubpath
	}
	out[key+".js"] = struct{}{}
	out[key+".d.ts"] = struct{}{}
	return out
}

// writeGeneratedDistModules writes dist/types, shared modules, and per-package core/pkg files.
func writeGeneratedDistModules(
	distDir, coreDir, pkgDir string,
	merged *transformerts.TypeScriptOutput,
	clientOutputs []*transformerts.TypeScriptOutput,
	genCfg ftconfig.GenerateConfig,
	runtime transformerts.ClientRuntime,
	invokePort string,
	log *logrus.Logger,
	stats *generateWriteStats,
) error {
	typesPath := filepath.Join(distDir, "types.d.ts")
	typesCode := emitTypesDTSFromMerged(merged)
	if err := writeGeneratedFile(typesPath, []byte(typesCode), stats); err != nil {
		return fmt.Errorf("failed to write types declaration file: %w", err)
	}
	log.WithFields(logrus.Fields{"path": typesPath}).Debug("Generated types declaration file")

	errorsJSPath := filepath.Join(distDir, transformerts.InfraErrorsSubpath+".js")
	errorsJS, err := transformerts.EmitErrorsESM(genCfg.PackageName, merged.DomainErrors, runtime)
	if err != nil {
		return fmt.Errorf("failed to emit errors.js: %w", err)
	}
	if err := writeGeneratedFile(errorsJSPath, []byte(errorsJS), stats); err != nil {
		return fmt.Errorf("failed to write errors.js: %w", err)
	}
	log.WithFields(logrus.Fields{"path": errorsJSPath}).Debug("Generated errors module")

	errorsDTSPath := filepath.Join(distDir, transformerts.InfraErrorsSubpath+".d.ts")
	errorsDTS, err := transformerts.EmitErrorsDTS(genCfg.PackageName, merged.DomainErrors, runtime)
	if err != nil {
		return fmt.Errorf("failed to emit errors.d.ts: %w", err)
	}
	if err := writeGeneratedFile(errorsDTSPath, []byte(errorsDTS), stats); err != nil {
		return fmt.Errorf("failed to write errors.d.ts: %w", err)
	}

	domainPackages, err := transformerts.BuildPackageDomainErrorEmits(genCfg.PackageName, clientOutputs)
	if err != nil {
		return fmt.Errorf("failed to build domain error decode metadata: %w", err)
	}

	transportJSPath := filepath.Join(distDir, "transport.js")
	if err := writeGeneratedFile(transportJSPath, []byte(transformerts.EmitTransportESM(invokePort, runtime, domainPackages)), stats); err != nil {
		return fmt.Errorf("failed to write transport.js: %w", err)
	}
	log.WithFields(logrus.Fields{"path": transportJSPath}).Debug("Generated transport module")

	transportDTSPath := filepath.Join(distDir, "transport.d.ts")
	if err := writeGeneratedFile(transportDTSPath, []byte(transformerts.EmitTransportDTS()), stats); err != nil {
		return fmt.Errorf("failed to write transport.d.ts: %w", err)
	}

	if runtime == transformerts.RuntimeEffect {
		effectJSPath := filepath.Join(distDir, transformerts.EffectModuleStem+".js")
		if err := writeGeneratedFile(effectJSPath, []byte(transformerts.EmitEffectSupportESM(genCfg.PackageName)), stats); err != nil {
			return fmt.Errorf("failed to write effect.js: %w", err)
		}
		effectDTSPath := filepath.Join(distDir, transformerts.EffectModuleStem+".d.ts")
		if err := writeGeneratedFile(effectDTSPath, []byte(transformerts.EmitEffectSupportDTS(genCfg.PackageName)), stats); err != nil {
			return fmt.Errorf("failed to write effect.d.ts: %w", err)
		}
		log.WithFields(logrus.Fields{"path": effectJSPath}).Debug("Generated Effect transport support module")
	}

	activePackages := make(map[string]struct{}, len(clientOutputs))
	for _, out := range clientOutputs {
		pkg := out.PackageName
		activePackages[pkg] = struct{}{}
		if len(out.DomainErrors) > 0 {
			pkgErrorsJS, err := transformerts.EmitPackageDomainErrorsESM(genCfg.PackageName, pkg, out.DomainErrors, runtime)
			if err != nil {
				return fmt.Errorf("failed to emit domain errors for package %s: %w", pkg, err)
			}
			pkgErrorsJSPath := filepath.Join(pkgDir, transformerts.PackageDomainErrorsFileStem(pkg)+".js")
			if err := writeGeneratedFile(pkgErrorsJSPath, []byte(pkgErrorsJS), stats); err != nil {
				return fmt.Errorf("failed to write package domain errors %s: %w", pkgErrorsJSPath, err)
			}
			pkgErrorsDTS, err := transformerts.EmitPackageDomainErrorsDTS(genCfg.PackageName, pkg, out.DomainErrors, runtime)
			if err != nil {
				return fmt.Errorf("failed to emit domain error declarations for package %s: %w", pkg, err)
			}
			pkgErrorsDTSPath := filepath.Join(pkgDir, transformerts.PackageDomainErrorsFileStem(pkg)+".d.ts")
			if err := writeGeneratedFile(pkgErrorsDTSPath, []byte(pkgErrorsDTS), stats); err != nil {
				return fmt.Errorf("failed to write package domain error declarations %s: %w", pkgErrorsDTSPath, err)
			}
			log.WithFields(logrus.Fields{
				"forstPackage": pkg,
				"path":         pkgErrorsJSPath,
			}).Debug("Generated package domain errors module")
		}

		mod := transformerts.ModuleEmitFromOutput(out, genCfg.OmitStubs)

		coreJS := filepath.Join(coreDir, pkg+".js")
		if err := writeGeneratedFile(coreJS, []byte(transformerts.EmitCoreESM(mod, invokePort)), stats); err != nil {
			return fmt.Errorf("failed to write core module %s: %w", coreJS, err)
		}
		coreDTS := filepath.Join(coreDir, pkg+".d.ts")
		if err := writeGeneratedFile(coreDTS, []byte(transformerts.EmitCoreDTS(mod)), stats); err != nil {
			return fmt.Errorf("failed to write core declarations %s: %w", coreDTS, err)
		}
		log.WithFields(logrus.Fields{
			"forstPackage":  pkg,
			"functionCount": len(out.Functions),
			"path":          coreJS,
		}).Debug("Generated core module")

		pkgJS := filepath.Join(pkgDir, pkg+".js")
		if err := writeGeneratedFile(pkgJS, []byte(transformerts.EmitPackageESM(mod, runtime, genCfg.PackageName)), stats); err != nil {
			return fmt.Errorf("failed to write package module %s: %w", pkgJS, err)
		}
		pkgDTS := filepath.Join(pkgDir, pkg+".d.ts")
		if err := writeGeneratedFile(pkgDTS, []byte(transformerts.EmitPackageDTS(mod, runtime, genCfg.PackageName)), stats); err != nil {
			return fmt.Errorf("failed to write package declarations %s: %w", pkgDTS, err)
		}
		log.WithFields(logrus.Fields{
			"forstPackage":  pkg,
			"functionCount": len(out.Functions),
			"path":          pkgJS,
			"runtime":       runtime.String(),
		}).Debug("Generated package module")
	}

	return nil
}

// emitTypesDTSFromMerged builds dist/types.d.ts from merged shapes (+ StreamingResult when needed).
func emitTypesDTSFromMerged(merged *transformerts.TypeScriptOutput) string {
	shapes := append([]string(nil), merged.Types...)
	sort.Strings(shapes)
	for _, fn := range merged.Functions {
		if fn.StreamingRowType != "" {
			shapes = append(shapes, transformerts.EmitStreamingResultTypeDeclaration())
			break
		}
	}
	return transformerts.EmitTypesDTS(shapes)
}

// printGenerateSummary writes the resolved specifier and one example import.
func printGenerateSummary(genCfg ftconfig.GenerateConfig, outputs []*transformerts.TypeScriptOutput) {
	pkgCount := len(outputs)
	fnCount := 0
	for _, out := range outputs {
		fnCount += len(out.Functions)
	}
	_, _ = fmt.Fprintf(generateReportWriter, "generate: wrote %s -> %s (%d packages, %d functions)\n",
		genCfg.PackageName, genCfg.OutDir, pkgCount, fnCount)
	if example, ok := exampleImportLine(genCfg.PackageName, outputs); ok {
		_, _ = fmt.Fprintf(generateReportWriter, "  %s\n", example)
	}
}

// exampleImportLine returns a sample named import from the first package with functions.
func exampleImportLine(packageName string, outputs []*transformerts.TypeScriptOutput) (string, bool) {
	for _, out := range outputs {
		if len(out.Functions) == 0 {
			continue
		}
		fn := out.Functions[0].Name
		return fmt.Sprintf(`import { %s } from "%s/%s"`, fn, packageName, out.PackageName), true
	}
	return "", false
}

// runnableClientOutputs returns package outputs that have public invoke exports.
func runnableClientOutputs(outputs []*transformerts.TypeScriptOutput) []*transformerts.TypeScriptOutput {
	var out []*transformerts.TypeScriptOutput
	for _, o := range outputs {
		if transformerts.PackageHasRunnableExports(o) {
			out = append(out, o)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].PackageName < out[j].PackageName
	})
	return out
}

// pruneStaleClientModules removes stale modules under dist/pkg/ and dist/core/.
func pruneStaleClientModules(distDir string, activePackages, activePackageErrors map[string]struct{}, testingSubpath string, log *logrus.Logger) error {
	for _, sub := range []string{"pkg", "core"} {
		if err := pruneStaleModulesInDir(filepath.Join(distDir, sub), activePackages, activePackageErrors, log); err != nil {
			return err
		}
	}
	reserved := reservedDistFileSet(testingSubpath)
	entries, err := generateIO.ReadDir(distDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, ok := reserved[name]; ok {
			continue
		}
		switch {
		case strings.HasSuffix(name, ".js"),
			strings.HasSuffix(name, ".d.ts"),
			strings.HasSuffix(name, ".ts"):
			path := filepath.Join(distDir, name)
			if err := generateIO.Remove(path); err != nil {
				return err
			}
			log.Debugf("Pruned stale client module: %s", path)
		}
	}
	return nil
}

func pruneStaleModulesInDir(dir string, activePackages, activePackageErrors map[string]struct{}, log *logrus.Logger) error {
	entries, err := generateIO.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		var pkg string
		var isDomainErrorsModule bool
		switch {
		case strings.HasSuffix(name, ".errors.d.ts"):
			pkg = strings.TrimSuffix(name, ".errors.d.ts")
			isDomainErrorsModule = true
		case strings.HasSuffix(name, ".errors.js"):
			pkg = strings.TrimSuffix(name, ".errors.js")
			isDomainErrorsModule = true
		case strings.HasSuffix(name, ".d.ts"):
			pkg = strings.TrimSuffix(name, ".d.ts")
		case strings.HasSuffix(name, ".js"):
			pkg = strings.TrimSuffix(name, ".js")
		case strings.HasSuffix(name, ".ts"):
			pkg = strings.TrimSuffix(name, ".ts")
		default:
			continue
		}
		active := activePackages
		if isDomainErrorsModule {
			active = activePackageErrors
		}
		if _, ok := active[pkg]; ok {
			continue
		}
		path := filepath.Join(dir, name)
		if err := generateIO.Remove(path); err != nil {
			return err
		}
		log.Debugf("Pruned stale client module: %s", path)
	}
	return nil
}
