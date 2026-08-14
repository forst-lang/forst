package main

import (
	"flag"
	"fmt"
	"forst/internal/ftconfig"
	transformerts "forst/internal/transformer/ts"
	"io"
	"os"
	"path/filepath"
	"time"

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
	configPath        string
	allowStemMismatch bool
	watch             bool
	listJSON          bool
	logLevel          string
	target            string
}

func parseGenerateArgs(args []string) (generateOptions, error) {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "Path to ftconfig.json")
	allowStemMismatch := fs.Bool("allow-stem-package-mismatch", false, "Allow .ft file stems that differ from declared package name")
	watch := fs.Bool("watch", false, "Regenerate when .ft files change")
	listJSON := fs.Bool("json", false, "With --list, emit a JSON manifest of packages and functions")
	list := fs.Bool("list", false, "Print a manifest of packages and functions instead of writing output")
	logLevel := fs.String("log-level", "", "Log level (trace, debug, info, warn, error)")
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
		listJSON:          *list || *listJSON,
		logLevel:          *logLevel,
		target:            tail[0],
	}, nil
}

func configureGenerateLogger(log *logrus.Logger, opts generateOptions, cfg *ForstConfig) {
	logLevel := opts.logLevel
	if logLevel == "" {
		logLevel = cfg.Dev.LogLevel
	}
	setLogLevel(log, logLevel)
}

func generateTSOptions(cfg *ForstConfig) *transformerts.GenerateTSOptions {
	return &transformerts.GenerateTSOptions{
		GenerateStreamingClients: cfg.Compiler.GenerateStreamingClients,
		ReportPhases:             cfg.Compiler.ReportPhases,
	}
}

func logGeneratePhase(log *logrus.Logger, reportPhases bool, phase string, start time.Time) {
	fields := logrus.Fields{
		"phase":      phase,
		"durationMs": time.Since(start).Milliseconds(),
	}
	if reportPhases {
		log.WithFields(fields).Info("generate phase")
	} else {
		log.WithFields(fields).Debug("generate phase")
	}
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
	configureGenerateLogger(log, opts, cfg)

	if opts.watch {
		return watchGenerate(opts, cfg, fileInfo.IsDir(), log)
	}

	if opts.listJSON {
		return runGenerateList(opts, cfg, fileInfo.IsDir(), log)
	}
	return runGenerateOnce(opts, cfg, fileInfo.IsDir(), log)
}

// runGenerateList discovers exports and prints a JSON manifest without writing client output.
func runGenerateList(opts generateOptions, cfg *ForstConfig, isDir bool, log *logrus.Logger) error {
	reportPhases := cfg.Compiler.ReportPhases

	discoverStart := time.Now()
	forstFiles, outputDir, err := discoverForstFilesForGenerate(cfg, opts.target, isDir)
	logGeneratePhase(log, reportPhases, "discover", discoverStart)
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
	if len(forstFiles) == 0 {
		log.Warn("No .ft files found for generation (check ftconfig include/exclude)")
		return printGenerateManifest(boundaryRoot, genCfg, nil, log)
	}
	if err := validateDiscoveredFileStemsHook(forstFiles, opts.allowStemMismatch, log); err != nil {
		return err
	}
	typecheckStart := time.Now()
	outputs, err := generateTSOutputsByPackageHook(forstFiles, log, generateTSOptions(cfg))
	logGeneratePhase(log, reportPhases, "typecheck_emit", typecheckStart)
	if err != nil {
		return err
	}
	return printGenerateManifest(boundaryRoot, genCfg, outputs, log)
}

// runGenerateOnce performs a single generate pass into outDir/dist/.
func runGenerateOnce(opts generateOptions, cfg *ForstConfig, isDir bool, log *logrus.Logger) error {
	reportPhases := cfg.Compiler.ReportPhases

	discoverStart := time.Now()
	forstFiles, outputDir, err := discoverForstFilesForGenerate(cfg, opts.target, isDir)
	logGeneratePhase(log, reportPhases, "discover", discoverStart)
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
	}).Debug("Resolved generate config")

	if len(forstFiles) == 0 {
		log.Warn("No .ft files found for generation (check ftconfig include/exclude)")
		return nil
	}

	log.Debugf("Found %d Forst files", len(forstFiles))

	if err := validateDiscoveredFileStemsHook(forstFiles, opts.allowStemMismatch, log); err != nil {
		return err
	}

	typecheckStart := time.Now()
	outputs, err := generateTSOutputsByPackageHook(forstFiles, log, generateTSOptions(cfg))
	logGeneratePhase(log, reportPhases, "typecheck_emit", typecheckStart)
	if err != nil {
		return err
	}
	reportProviderOmissions(outputs, log)

	// Guards run before any emit so a failing project leaves no partial output.
	packageNames := transformerts.PackageNames(outputs)
	if err := transformerts.ValidateForstPackageNames(packageNames); err != nil {
		log.Error(err.Error())
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
	if err := transformerts.ValidateDomainErrors(merged.DomainErrors); err != nil {
		log.Error(err.Error())
		return err
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

	writeStart := time.Now()
	clientOutputs := runnableClientOutputs(outputs)
	if err := writeGeneratedDistModules(distDir, coreDir, pkgDir, merged, clientOutputs, genCfg, runtime, invokePort, log, &stats); err != nil {
		return err
	}

	activePackages := make(map[string]struct{}, len(clientOutputs))
	activePackageErrors := make(map[string]struct{}, len(clientOutputs))
	for _, out := range clientOutputs {
		activePackages[out.PackageName] = struct{}{}
		if len(out.DomainErrors) > 0 {
			activePackageErrors[out.PackageName] = struct{}{}
		}
	}
	if err := pruneStaleClientModulesHook(distDir, activePackages, activePackageErrors, genCfg.TestingSubpath, log); err != nil {
		return fmt.Errorf("prune stale client modules: %w", err)
	}

	if err := generateClientPackageHook(outDir, genCfg, clientOutputs, activePackageErrors, invokePort, log, &stats); err != nil {
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
	logGeneratePhase(log, reportPhases, "write", writeStart)

	linkStart := time.Now()
	if err := maybeLinkGeneratedClient(boundaryRoot, outDir, genCfg, log); err != nil {
		return err
	}
	logGeneratePhase(log, reportPhases, "link", linkStart)
	warnMissingLifecycleScript(boundaryRoot, genCfg, log)

	printGenerateSummary(genCfg, clientOutputs)

	log.WithFields(logrus.Fields{
		"filesWritten": stats.Written,
		"filesSkipped": stats.Skipped,
	}).Debug("Generate write summary")

	log.Debug("TypeScript client generation completed")
	return nil
}
