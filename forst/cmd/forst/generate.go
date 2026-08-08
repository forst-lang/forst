package main

import (
	"flag"
	"fmt"
	"forst/internal/ftconfig"
	transformerts "forst/internal/transformer/ts"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// generateIO hooks filesystem operations for tests.
var generateIO = struct {
	MkdirAll  func(string, os.FileMode) error
	WriteFile func(string, []byte, os.FileMode) error
	ReadFile  func(string) ([]byte, error)
	Remove    func(string) error
	ReadDir   func(string) ([]os.DirEntry, error)
	Rename    func(oldpath, newpath string) error
}{
	MkdirAll:  os.MkdirAll,
	WriteFile: os.WriteFile,
	ReadFile:  os.ReadFile,
	Remove:    os.Remove,
	ReadDir:   os.ReadDir,
	Rename:    os.Rename,
}

// generateReportWriter receives the post-emit specifier summary (defaults to stdout).
var generateReportWriter io.Writer = os.Stdout

var (
	absPathForGenerate              = filepath.Abs
	mergeTypeScriptOutputsHook      = transformerts.MergeTypeScriptOutputs
	generateTSOutputsByPackageHook  = transformerts.GenerateTypeScriptOutputsByPackage
	validateDiscoveredFileStemsHook = transformerts.ValidateDiscoveredFileStems
	generateClientPackageHook       = generateClientPackage
	pruneStaleClientModulesHook     = pruneStaleClientModules
	newGenerateLogger               = func() *logrus.Logger {
		log := logrus.New()
		log.SetLevel(logrus.InfoLevel)
		return log
	}
)

// reservedDistFiles are never pruned as stale package modules under outDir/dist/.
var reservedDistFiles = map[string]struct{}{
	"index.js":       {},
	"index.d.ts":     {},
	"transport.js":   {},
	"transport.d.ts": {},
	"types.d.ts":     {},
	"errors.js":      {},
	"errors.d.ts":    {},
	"effect.js":      {},
	"effect.d.ts":    {},
	"testing.js":     {},
	"testing.d.ts":   {},
}

// reservedDistFileSet returns compiler-owned dist root files, including the configured testing subpath.
func reservedDistFileSet(testingSubpath string) map[string]struct{} {
	out := make(map[string]struct{}, len(reservedDistFiles)+2)
	for k, v := range reservedDistFiles {
		out[k] = v
	}
	key := testingSubpath
	if key == "" {
		key = "testing"
	}
	out[key+".js"] = struct{}{}
	out[key+".d.ts"] = struct{}{}
	return out
}

// loadConfigForGenerate resolves ftconfig: explicit -config, else search upward from target, else defaults.
func loadConfigForGenerate(explicitConfig string, target string, isDir bool) (*ForstConfig, error) {
	if explicitConfig != "" {
		abs, err := absPathForGenerate(explicitConfig)
		if err != nil {
			return nil, err
		}
		return LoadConfig(abs)
	}
	startDir := target
	if !isDir {
		startDir = filepath.Dir(target)
	}
	abs, err := absPathForGenerate(startDir)
	if err != nil {
		return nil, err
	}
	found, _ := FindConfigFile(abs)
	if found != "" {
		return LoadConfig(found)
	}
	return DefaultConfig(), nil
}

// discoverForstFilesForGenerate lists .ft files using the same include/exclude rules as `forst dev`.
func discoverForstFilesForGenerate(cfg *ForstConfig, target string, isDir bool) (forstFiles []string, outputDir string, err error) {
	if isDir {
		absTarget, err := absPathForGenerate(target)
		if err != nil {
			return nil, "", err
		}
		forstFiles, err = cfg.FindForstFiles(absTarget)
		if err != nil {
			return nil, "", err
		}
		return forstFiles, absTarget, nil
	}
	if filepath.Ext(target) != ".ft" {
		return nil, "", fmt.Errorf("target file must have .ft extension")
	}
	absFile, err := absPathForGenerate(target)
	if err != nil {
		return nil, "", err
	}
	dir := filepath.Dir(absFile)
	candidates, err := cfg.FindForstFiles(dir)
	if err != nil {
		return nil, "", err
	}
	for _, f := range candidates {
		if filepath.Clean(f) == filepath.Clean(absFile) {
			return []string{absFile}, dir, nil
		}
	}
	return nil, "", fmt.Errorf("file %s is not included by ftconfig discovery rules (include/exclude)", target)
}

type generateOptions struct {
	configPath         string
	allowStemMismatch  bool
	watch              bool
	target             string
}

func parseGenerateArgs(args []string) (generateOptions, error) {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "Path to ftconfig.json")
	allowStemMismatch := fs.Bool("allow-stem-package-mismatch", false, "Allow .ft file stems that differ from declared package name")
	watch := fs.Bool("watch", false, "Regenerate when .ft files change")
	if err := fs.Parse(args); err != nil {
		return generateOptions{}, err
	}
	tail := fs.Args()
	if len(tail) < 1 {
		return generateOptions{}, fmt.Errorf("generate command requires a target file or directory")
	}
	return generateOptions{
		configPath:        *configPath,
		allowStemMismatch: *allowStemMismatch,
		watch:             *watch,
		target:            tail[0],
	}, nil
}

// generateCommand handles the "forst generate" command
func generateCommand(args []string) error {
	opts, err := parseGenerateArgs(args)
	if err != nil {
		return err
	}

	log := newGenerateLogger()

	fileInfo, err := os.Stat(opts.target)
	if err != nil {
		return fmt.Errorf("failed to stat target %s: %w", opts.target, err)
	}

	cfg, err := loadConfigForGenerate(opts.configPath, opts.target, fileInfo.IsDir())
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if opts.watch {
		return watchGenerate(opts, cfg, fileInfo.IsDir(), log)
	}
	return runGenerateOnce(opts, cfg, fileInfo.IsDir(), log)
}

// runGenerateOnce performs a single generate pass into outDir/dist/.
func runGenerateOnce(opts generateOptions, cfg *ForstConfig, isDir bool, log *logrus.Logger) error {
	forstFiles, outputDir, err := discoverForstFilesForGenerate(cfg, opts.target, isDir)
	if err != nil {
		return err
	}

	boundaryRoot := outputDir
	if root, rootErr := ftconfig.BoundaryRootFromDir(outputDir); rootErr == nil {
		boundaryRoot = root
	}
	genCfg := ftconfig.EffectiveGenerateConfig(&cfg.Config, boundaryRoot)
	if err := genCfg.Validate(); err != nil {
		return fmt.Errorf("generate config: %w", err)
	}
	log.WithFields(logrus.Fields{
		"packageName": genCfg.PackageName,
		"outDir":      genCfg.OutDir,
		"ephemeral":   genCfg.IsEphemeral(boundaryRoot),
		"link":        genCfg.ShouldLink(boundaryRoot),
		"emit":        genCfg.Emit,
		"effect":      genCfg.Effect,
	}).Info("Resolved generate config")

	if len(forstFiles) == 0 {
		log.Warn("No .ft files found for generation (check ftconfig include/exclude)")
		return nil
	}

	log.Infof("Found %d Forst files", len(forstFiles))

	if err := validateDiscoveredFileStemsHook(forstFiles, opts.allowStemMismatch, log); err != nil {
		return err
	}

	outputs, err := generateTSOutputsByPackageHook(forstFiles, log, &transformerts.GenerateTSOptions{
		GenerateStreamingClients: cfg.Compiler.GenerateStreamingClients,
	})
	if err != nil {
		return err
	}
	reportProviderOmissions(outputs, log)

	// Guards run before any emit so a failing project leaves no partial output.
	packageNames := transformerts.PackageNames(outputs)
	if err := transformerts.ValidateReservedSubpaths(packageNames, genCfg.ReservedSubpaths()); err != nil {
		log.WithFields(logrus.Fields{
			"reserved": transformerts.FormatReservedSubpathKeys(genCfg.ReservedSubpaths()),
		}).Error(err.Error())
		return err
	}
	runtime := transformerts.RuntimeFromConfig(genCfg)
	if runtime == transformerts.RuntimeEffect {
		if err := transformerts.ValidateServiceClassNames(packageNames); err != nil {
			log.Error(err.Error())
			return err
		}
		if err := requireEffectRuntime(boundaryRoot); err != nil {
			log.Error(err.Error())
			return err
		}
	}

	merged, err := mergeTypeScriptOutputsHook(outputs)
	if err != nil {
		log.WithError(err).Error("Type name conflict while merging TypeScript outputs")
		return fmt.Errorf("merge TypeScript outputs: %w", err)
	}

	outDir := genCfg.EffectiveOutDir(boundaryRoot)
	distDir := filepath.Join(outDir, "dist")
	coreDir := filepath.Join(distDir, "core")
	pkgDir := filepath.Join(distDir, "pkg")
	invokePort := cfg.Server.EffectiveInvokePort()

	if err := generateIO.MkdirAll(coreDir, 0755); err != nil {
		return fmt.Errorf("failed to create dist/core directory: %w", err)
	}
	if err := generateIO.MkdirAll(pkgDir, 0755); err != nil {
		return fmt.Errorf("failed to create dist/pkg directory: %w", err)
	}

	var stats generateWriteStats

	typesPath := filepath.Join(distDir, "types.d.ts")
	typesCode := emitTypesDTSFromMerged(merged)
	if err := writeGeneratedFile(typesPath, []byte(typesCode), &stats); err != nil {
		return fmt.Errorf("failed to write types declaration file: %w", err)
	}
	log.WithFields(logrus.Fields{"path": typesPath}).Info("Generated types declaration file")

	errorsJSPath := filepath.Join(distDir, "errors.js")
	if err := writeGeneratedFile(errorsJSPath, []byte(transformerts.EmitErrorsESM()), &stats); err != nil {
		return fmt.Errorf("failed to write errors.js: %w", err)
	}
	log.WithFields(logrus.Fields{"path": errorsJSPath}).Info("Generated errors module")

	errorsDTSPath := filepath.Join(distDir, "errors.d.ts")
	if err := writeGeneratedFile(errorsDTSPath, []byte(transformerts.EmitErrorsDTS()), &stats); err != nil {
		return fmt.Errorf("failed to write errors.d.ts: %w", err)
	}

	transportJSPath := filepath.Join(distDir, "transport.js")
	if err := writeGeneratedFile(transportJSPath, []byte(transformerts.EmitTransportESM(invokePort)), &stats); err != nil {
		return fmt.Errorf("failed to write transport.js: %w", err)
	}
	log.WithFields(logrus.Fields{"path": transportJSPath}).Info("Generated transport module")

	transportDTSPath := filepath.Join(distDir, "transport.d.ts")
	if err := writeGeneratedFile(transportDTSPath, []byte(transformerts.EmitTransportDTS()), &stats); err != nil {
		return fmt.Errorf("failed to write transport.d.ts: %w", err)
	}

	if runtime == transformerts.RuntimeEffect {
		effectJSPath := filepath.Join(distDir, transformerts.EffectModuleStem+".js")
		if err := writeGeneratedFile(effectJSPath, []byte(transformerts.EmitEffectSupportESM(genCfg.PackageName)), &stats); err != nil {
			return fmt.Errorf("failed to write effect.js: %w", err)
		}
		effectDTSPath := filepath.Join(distDir, transformerts.EffectModuleStem+".d.ts")
		if err := writeGeneratedFile(effectDTSPath, []byte(transformerts.EmitEffectSupportDTS(genCfg.PackageName)), &stats); err != nil {
			return fmt.Errorf("failed to write effect.d.ts: %w", err)
		}
		log.WithFields(logrus.Fields{"path": effectJSPath}).Info("Generated Effect transport support module")
	}

	clientOutputs := runnableClientOutputs(outputs)
	activePackages := make(map[string]struct{}, len(clientOutputs))
	for _, out := range clientOutputs {
		pkg := out.PackageName
		activePackages[pkg] = struct{}{}
		mod := moduleEmitFromOutput(out, genCfg.OmitStubs)

		coreJS := filepath.Join(coreDir, pkg+".js")
		if err := writeGeneratedFile(coreJS, []byte(transformerts.EmitCoreESM(mod, invokePort)), &stats); err != nil {
			log.Errorf("Failed to write core module %s: %v", coreJS, err)
			continue
		}
		coreDTS := filepath.Join(coreDir, pkg+".d.ts")
		if err := writeGeneratedFile(coreDTS, []byte(transformerts.EmitCoreDTS(mod)), &stats); err != nil {
			log.Errorf("Failed to write core declarations %s: %v", coreDTS, err)
			continue
		}
		log.WithFields(logrus.Fields{
			"forstPackage":  pkg,
			"functionCount": len(out.Functions),
			"path":          coreJS,
		}).Info("Generated core module")

		pkgJS := filepath.Join(pkgDir, pkg+".js")
		if err := writeGeneratedFile(pkgJS, []byte(transformerts.EmitPackageESM(mod, runtime, genCfg.PackageName)), &stats); err != nil {
			log.Errorf("Failed to write package module %s: %v", pkgJS, err)
			continue
		}
		pkgDTS := filepath.Join(pkgDir, pkg+".d.ts")
		if err := writeGeneratedFile(pkgDTS, []byte(transformerts.EmitPackageDTS(mod, runtime, genCfg.PackageName)), &stats); err != nil {
			log.Errorf("Failed to write package declarations %s: %v", pkgDTS, err)
			continue
		}
		log.WithFields(logrus.Fields{
			"forstPackage":  pkg,
			"functionCount": len(out.Functions),
			"path":          pkgJS,
			"runtime":       runtime.String(),
		}).Info("Generated package module")
	}

	if err := pruneStaleClientModulesHook(distDir, activePackages, genCfg.TestingSubpath, log); err != nil {
		return fmt.Errorf("prune stale client modules: %w", err)
	}

	if err := generateClientPackageHook(outDir, genCfg, clientOutputs, invokePort, log, &stats); err != nil {
		return fmt.Errorf("generate client package: %w", err)
	}

	if genCfg.SSRModule != "" {
		if err := writeSSRModule(boundaryRoot, genCfg.SSRModule, outDir, clientOutputs, invokePort, log, &stats); err != nil {
			return err
		}
	}

	if err := writeForstGeneratedMarker(outDir, boundaryRoot, genCfg.PackageName, &stats); err != nil {
		return err
	}

	if err := maybeLinkGeneratedClient(boundaryRoot, outDir, genCfg, log); err != nil {
		return err
	}
	warnMissingLifecycleScript(boundaryRoot, genCfg, log)

	printGenerateSummary(genCfg, clientOutputs)

	log.WithFields(logrus.Fields{
		"filesWritten": stats.Written,
		"filesSkipped": stats.Skipped,
	}).Info("Generate write summary")
	log.WithFields(logrus.Fields{
		"filesWritten": stats.Written,
		"filesSkipped": stats.Skipped,
	}).Debug("Generate write summary (debug)")

	log.Info("TypeScript client generation completed")
	return nil
}

// moduleEmitFromOutput maps a transformer package output to Phase 3 emit input.
// When omitStubs is true, provider-gated omissions are attached for commented stubs.
func moduleEmitFromOutput(out *transformerts.TypeScriptOutput, omitStubs bool) transformerts.ModuleEmit {
	pkg := out.PackageName
	if pkg == "" {
		pkg = out.SourceFileStem
	}
	mod := transformerts.ModuleEmit{
		PackageName: pkg,
		Functions:   out.Functions,
		TypeImports: append([]string(nil), out.ExportedTypeNames...),
	}
	if omitStubs && len(out.OmittedFunctions) > 0 {
		mod.Omitted = append([]transformerts.OmittedFunction(nil), out.OmittedFunctions...)
	}
	return mod
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
	fmt.Fprintf(generateReportWriter, "generate: wrote %s -> %s (%d packages, %d functions)\n",
		genCfg.PackageName, genCfg.OutDir, pkgCount, fnCount)
	if example, ok := exampleImportLine(genCfg.PackageName, outputs); ok {
		fmt.Fprintf(generateReportWriter, "  %s\n", example)
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
func pruneStaleClientModules(distDir string, activePackages map[string]struct{}, testingSubpath string, log *logrus.Logger) error {
	for _, sub := range []string{"pkg", "core"} {
		if err := pruneStaleModulesInDir(filepath.Join(distDir, sub), activePackages, log); err != nil {
			return err
		}
	}
	reserved := reservedDistFileSet(testingSubpath)
	// Remove leftover non-reserved files at dist/ root (flat modules from older layouts).
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
			log.Infof("Pruned stale client module: %s", path)
		}
	}
	return nil
}

func pruneStaleModulesInDir(dir string, activePackages map[string]struct{}, log *logrus.Logger) error {
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
		switch {
		case strings.HasSuffix(name, ".d.ts"):
			pkg = strings.TrimSuffix(name, ".d.ts")
		case strings.HasSuffix(name, ".js"):
			pkg = strings.TrimSuffix(name, ".js")
		case strings.HasSuffix(name, ".ts"):
			pkg = strings.TrimSuffix(name, ".ts")
		default:
			continue
		}
		if _, ok := activePackages[pkg]; ok {
			continue
		}
		path := filepath.Join(dir, name)
		if err := generateIO.Remove(path); err != nil {
			return err
		}
		log.Infof("Pruned stale client module: %s", path)
	}
	return nil
}

// generateClientPackage writes package.json, dist/index.{js,d.ts}, and README.md under outDir.
func generateClientPackage(
	outDir string,
	genCfg ftconfig.GenerateConfig,
	outputs []*transformerts.TypeScriptOutput,
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

	indexJS := transformerts.EmitIndexESM(packageNames, invokePort)
	if runtime == transformerts.RuntimeEffect {
		indexJS += transformerts.EmitIndexEffectESM(packageNames, genCfg.PackageName)
	}
	indexJSPath := filepath.Join(outDir, "dist", "index.js")
	if err := writeGeneratedFile(indexJSPath, []byte(indexJS), stats); err != nil {
		return fmt.Errorf("failed to write client index.js: %w", err)
	}
	log.Infof("Generated client index: %s", indexJSPath)

	indexDTS := transformerts.EmitIndexDTS(packageNames)
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
		modules = append(modules, moduleEmitFromOutput(out, false))
	}
	testingKey := genCfg.TestingSubpath
	if testingKey == "" {
		testingKey = "testing"
	}
	var testingJS string
	var testingDTS string
	if runtime == transformerts.RuntimeEffect {
		testingJS = transformerts.EmitTestingEffectESM(modules)
		testingDTS = transformerts.EmitTestingEffectDTS(modules)
	} else {
		testingJS = transformerts.EmitTestingESM(modules)
		testingDTS = transformerts.EmitTestingDTS(modules)
	}
	testingJSPath := filepath.Join(outDir, "dist", testingKey+".js")
	if err := writeGeneratedFile(testingJSPath, []byte(testingJS), stats); err != nil {
		return fmt.Errorf("failed to write testing.js: %w", err)
	}
	log.Infof("Generated testing module: %s", testingJSPath)

	testingDTSPath := filepath.Join(outDir, "dist", testingKey+".d.ts")
	if err := writeGeneratedFile(testingDTSPath, []byte(testingDTS), stats); err != nil {
		return fmt.Errorf("failed to write testing.d.ts: %w", err)
	}

	packageContent := generateClientPackageJSON(genCfg, packageNames)
	packagePath := filepath.Join(outDir, "package.json")
	if err := writeGeneratedFile(packagePath, []byte(packageContent), stats); err != nil {
		return fmt.Errorf("failed to write client package.json: %w", err)
	}
	log.Infof("Generated client package.json: %s", packagePath)

	readme := generateClientREADME(genCfg, invokePort, outputs)
	readmePath := filepath.Join(outDir, "README.md")
	if err := writeGeneratedFile(readmePath, []byte(readme), stats); err != nil {
		return fmt.Errorf("failed to write client README: %w", err)
	}
	log.Infof("Generated client README: %s", readmePath)

	return nil
}

// writeSSRModule writes the configured SSR invoke surface relative to boundaryRoot.
// Imports use paths into outDir/dist (inlined transport), never @forst/client.
func writeSSRModule(
	boundaryRoot, ssrRelPath, outDir string,
	outputs []*transformerts.TypeScriptOutput,
	invokePort string,
	log *logrus.Logger,
	stats *generateWriteStats,
) error {
	modulePath := filepath.Join(boundaryRoot, filepath.Clean(ssrRelPath))
	if err := generateIO.MkdirAll(filepath.Dir(modulePath), 0755); err != nil {
		return fmt.Errorf("failed to create SSR module directory: %w", err)
	}

	distDir := filepath.Join(outDir, "dist")
	moduleDir := filepath.Dir(modulePath)
	transportImport, err := relativeJSImport(moduleDir, filepath.Join(distDir, "transport"))
	if err != nil {
		return fmt.Errorf("SSR transport import path: %w", err)
	}
	typesImport, err := relativeJSImport(moduleDir, filepath.Join(distDir, "types"))
	if err != nil {
		return fmt.Errorf("SSR types import path: %w", err)
	}

	sorted := append([]*transformerts.TypeScriptOutput(nil), outputs...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].PackageName < sorted[j].PackageName
	})

	var body strings.Builder
	body.WriteString("// Auto-generated Forst SSR invoke surface\n")
	body.WriteString("// Generated by forst generate — import from the configured generate.ssrModule path\n")
	body.WriteString(strings.TrimSuffix(transformerts.GeneratedHeaderComment(invokePort), "\n"))
	body.WriteString("\n\n")
	fmt.Fprintf(&body, "import { getDefaultInvokeClient } from '%s';\n", transportImport)
	typeNames := transformerts.CollectInvokeTypeNames(sorted)
	if len(typeNames) > 0 {
		fmt.Fprintf(&body, "import type { %s } from '%s';\n", strings.Join(typeNames, ", "), typesImport)
	}
	body.WriteString("\n")
	for _, out := range sorted {
		if len(out.Functions) == 0 {
			continue
		}
		for _, line := range transformerts.DirectInvokeExportLines(out.PackageName, out.Functions) {
			body.WriteString(line)
			body.WriteString("\n\n")
		}
	}

	if err := writeGeneratedFile(modulePath, []byte(body.String()), stats); err != nil {
		return fmt.Errorf("failed to write SSR invoke module: %w", err)
	}
	log.WithFields(logrus.Fields{
		"ssrModule": ssrRelPath,
		"path":      modulePath,
	}).Info("Generated SSR invoke module")
	return nil
}

// relativeJSImport returns a relative ESM specifier from fromDir to toPathWithoutExt (.js suffix).
func relativeJSImport(fromDir, toPathWithoutExt string) (string, error) {
	rel, err := filepath.Rel(fromDir, toPathWithoutExt)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}
	return rel + ".js", nil
}

// generateClientPackageJSON creates package.json with an exports map pointing at dist/.
func generateClientPackageJSON(genCfg ftconfig.GenerateConfig, packages []string) string {
	name := genCfg.PackageName
	if name == "" {
		name = ftconfig.DefaultPackageName
	}
	sorted := append([]string(nil), packages...)
	sort.Strings(sorted)
	seen := make(map[string]struct{}, len(sorted))
	var unique []string
	for _, pkg := range sorted {
		if pkg == "" {
			continue
		}
		if _, ok := seen[pkg]; ok {
			continue
		}
		seen[pkg] = struct{}{}
		unique = append(unique, pkg)
	}

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
	appendTestingPackageJSONExport(&b, genCfg.TestingSubpath)
	if genCfg.Effect {
		appendPackageJSONExport(&b, "./effect", "./dist/effect.d.ts", "./dist/effect.js")
	}
	b.WriteString("\n  }")
	if genCfg.Effect {
		b.WriteString(",\n")
		b.WriteString("  \"peerDependencies\": {\n")
		fmt.Fprintf(&b, "    \"effect\": %s\n", jsonString(transformerts.EffectPeerDependencyRange))
		b.WriteString("  }")
	}
	b.WriteString("\n}\n")
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

// appendTestingPackageJSONExport adds the reserved testing subpath export.
// Kept separate so Phase 4 package.json edits do not bury the testing key.
func appendTestingPackageJSONExport(b *strings.Builder, testingSubpath string) {
	key := testingSubpath
	if key == "" {
		key = "testing"
	}
	appendPackageJSONExport(b, "./"+key, "./dist/"+key+".d.ts", "./dist/"+key+".js")
}

func jsonString(s string) string {
	return strconv.Quote(s)
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
	b.WriteString("## Lifecycle script\n\n")
	b.WriteString("Ephemeral output under `.forst/` is gitignored. Add a postinstall script so fresh checkouts regenerate and relink:\n\n")
	b.WriteString("```json\n")
	b.WriteString(`{ "scripts": { "postinstall": "forst generate ." } }`)
	b.WriteString("\n```\n")
	if genCfg.Effect {
		b.WriteString("\n## Effect mode\n\n")
		b.WriteString("This package was generated with `generate.effect: true`.\n\n")
		fmt.Fprintf(&b, "- Peer dependency: `effect` %s (required for `Layer.mock`).\n", transformerts.EffectPeerDependencyRange)
		b.WriteString("- Call sites return `Effect.Effect<Response, InvokeFailure, PkgService>` and need `Effect.provide(ForstClientLive)` (or `layerForstClient`).\n")
		b.WriteString("- Mocking: `Layer.mock(Pkg, { ... })` for one service, `layerForstTest(overrides)` for the whole client, or `Layer.mock(ForstTransport, { client })` for the wire.\n")
		b.WriteString("- Tagged invoke errors match Promise mode. They carry `_tag` for `Effect.catchTag` but are not `Data.TaggedError`, so `Equal.equals` compares by reference.\n")
		b.WriteString("- Prefer `Effect.retry` over `options.retries` (omitted in Effect mode).\n")
	}
	return b.String()
}
