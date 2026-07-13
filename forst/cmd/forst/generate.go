package main

import (
	"flag"
	"fmt"
	transformerts "forst/internal/transformer/ts"
	"io"
	"os"
	"path/filepath"
	"sort"
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
}{
	MkdirAll:  os.MkdirAll,
	WriteFile: os.WriteFile,
	ReadFile:  os.ReadFile,
	Remove:    os.Remove,
	ReadDir:   os.ReadDir,
}

var (
	absPathForGenerate              = filepath.Abs
	mergeTypeScriptOutputsHook      = transformerts.MergeTypeScriptOutputs
	generateTSOutputsByPackageHook  = transformerts.GenerateTypeScriptOutputsByPackage
	validateDiscoveredFileStemsHook = transformerts.ValidateDiscoveredFileStems
	generateClientPackageHook       = generateClientPackage
	pruneStaleClientModulesHook     = pruneStaleClientModules
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

// generateCommand handles the "forst generate" command
func generateCommand(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "Path to ftconfig.json")
	allowStemMismatch := fs.Bool("allow-stem-package-mismatch", false, "Allow .ft file stems that differ from declared package name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	tail := fs.Args()
	if len(tail) < 1 {
		return fmt.Errorf("generate command requires a target file or directory")
	}

	target := tail[0]

	// Create logger
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)

	// Check if target is a file or directory
	fileInfo, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("failed to stat target %s: %w", target, err)
	}

	cfg, err := loadConfigForGenerate(*configPath, target, fileInfo.IsDir())
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	forstFiles, outputDir, err := discoverForstFilesForGenerate(cfg, target, fileInfo.IsDir())
	if err != nil {
		return err
	}

	if len(forstFiles) == 0 {
		log.Warn("No .ft files found for generation (check ftconfig include/exclude)")
		return nil
	}

	log.Infof("Found %d Forst files", len(forstFiles))

	if err := validateDiscoveredFileStemsHook(forstFiles, *allowStemMismatch, log); err != nil {
		return err
	}

	outputs, err := generateTSOutputsByPackageHook(forstFiles, log, &transformerts.GenerateTSOptions{
		GenerateStreamingClients: cfg.Compiler.GenerateStreamingClients,
	})
	if err != nil {
		return err
	}

	merged, err := mergeTypeScriptOutputsHook(outputs)
	if err != nil {
		return fmt.Errorf("merge TypeScript outputs: %w", err)
	}

	generatedDir := filepath.Join(outputDir, "generated")
	if err := generateIO.MkdirAll(generatedDir, 0755); err != nil {
		return fmt.Errorf("failed to create generated directory: %w", err)
	}

	typesPath := filepath.Join(generatedDir, "types.d.ts")
	typesCode := merged.GenerateTypesFile()
	if err := generateIO.WriteFile(typesPath, []byte(typesCode), 0644); err != nil {
		return fmt.Errorf("failed to write types declaration file: %w", err)
	}
	log.Infof("Generated types declaration file: %s", typesPath)

	clientOutputs := runnableClientOutputs(outputs)
	activePackages := make(map[string]struct{}, len(clientOutputs))
	for _, out := range clientOutputs {
		pkg := out.PackageName
		activePackages[pkg] = struct{}{}
		clientPath := filepath.Join(generatedDir, pkg+".client.ts")
		clientCode := out.GenerateClientFile()
		if err := generateIO.WriteFile(clientPath, []byte(clientCode), 0644); err != nil {
			log.Errorf("Failed to write client module %s: %v", clientPath, err)
			continue
		}
		log.Infof("Generated client module: %s", clientPath)
	}

	if err := pruneStaleClientModulesHook(generatedDir, activePackages, log); err != nil {
		return fmt.Errorf("prune stale client modules: %w", err)
	}

	// Generate client package structure (only packages with runnable exports)
	if err := generateClientPackageHook(outputDir, clientOutputs, log); err != nil {
		log.Errorf("Failed to generate client package: %v", err)
	}

	log.Info("TypeScript declaration files and client implementations generation completed")
	return nil
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

// pruneStaleClientModules removes generated/*.client.ts files for packages no longer emitted.
func pruneStaleClientModules(generatedDir string, activePackages map[string]struct{}, log *logrus.Logger) error {
	entries, err := generateIO.ReadDir(generatedDir)
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
		if !strings.HasSuffix(name, ".client.ts") {
			continue
		}
		pkg := strings.TrimSuffix(name, ".client.ts")
		if _, ok := activePackages[pkg]; ok {
			continue
		}
		path := filepath.Join(generatedDir, name)
		if err := generateIO.Remove(path); err != nil {
			return err
		}
		log.Infof("Pruned stale client module: %s", path)
	}
	return nil
}

// generateClientPackage creates the main client package structure
func generateClientPackage(outputDir string, outputs []*transformerts.TypeScriptOutput, log *logrus.Logger) error {
	// Create client package directory
	clientDir := filepath.Join(outputDir, "client")
	if err := generateIO.MkdirAll(clientDir, 0755); err != nil {
		return fmt.Errorf("failed to create client directory: %w", err)
	}

	// Generate main client index file
	indexContent := generateClientIndex(outputs)
	indexPath := filepath.Join(clientDir, "index.ts")
	if err := generateIO.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
		return fmt.Errorf("failed to write client index: %w", err)
	}
	log.Infof("Generated client index: %s", indexPath)

	// Generate package.json for the client
	packageContent := generateClientPackageJSON()
	packagePath := filepath.Join(clientDir, "package.json")
	if err := generateIO.WriteFile(packagePath, []byte(packageContent), 0644); err != nil {
		return fmt.Errorf("failed to write client package.json: %w", err)
	}
	log.Infof("Generated client package.json: %s", packagePath)

	// Copy types declaration file to client directory
	typesSource := filepath.Join(outputDir, "generated", "types.d.ts")
	typesDest := filepath.Join(clientDir, "types.d.ts")
	if err := copyFile(typesSource, typesDest); err != nil {
		return fmt.Errorf("failed to copy types declaration file: %w", err)
	}
	log.Infof("Copied types declaration file to client directory")

	if err := generateSSRInvokeModule(outputDir, outputs, log); err != nil {
		return err
	}

	return nil
}

// generateSSRInvokeModule writes app/lib/forst.invoke.ts when app/lib exists.
// Remix/Vite only bundles modules under app/ — this gives SSR loaders a stable surface.
func generateSSRInvokeModule(outputDir string, outputs []*transformerts.TypeScriptOutput, log *logrus.Logger) error {
	appLibDir := filepath.Join(outputDir, "app", "lib")
	if st, err := os.Stat(appLibDir); err != nil || !st.IsDir() {
		return nil
	}

	sorted := append([]*transformerts.TypeScriptOutput(nil), outputs...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].PackageName < sorted[j].PackageName
	})

	var body strings.Builder
	body.WriteString(`// Auto-generated Forst SSR invoke surface
// Generated by forst generate — import from ../lib/forst.invoke in Remix loaders
// Embedded invoke: set FORST_BASE_URL=http://127.0.0.1:8081

import { getDefaultInvokeClient } from '@forst/client';
`)
	typeNames := transformerts.CollectInvokeTypeNames(sorted)
	if len(typeNames) > 0 {
		fmt.Fprintf(&body, "import type { %s } from '../../generated/types';\n", strings.Join(typeNames, ", "))
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

	modulePath := filepath.Join(appLibDir, "forst.invoke.ts")
	if err := generateIO.WriteFile(modulePath, []byte(body.String()), 0644); err != nil {
		return fmt.Errorf("failed to write SSR invoke module: %w", err)
	}
	log.Infof("Generated SSR invoke module: %s", modulePath)
	return nil
}

// generateClientIndex creates the main client index file.
func generateClientIndex(outputs []*transformerts.TypeScriptOutput) string {
	sorted := append([]*transformerts.TypeScriptOutput(nil), outputs...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].PackageName < sorted[j].PackageName
	})

	var imports []string
	var exports []string
	var packageProperties []string
	var reexports strings.Builder

	for _, out := range sorted {
		baseName := out.PackageName
		imports = append(imports, fmt.Sprintf("import { %s } from '../generated/%s.client';", baseName, baseName))
		exports = append(exports, baseName)
		packageProperties = append(packageProperties, fmt.Sprintf("  public %s: ReturnType<typeof %s>;", baseName, baseName))
		names := make([]string, 0, len(out.Functions)+1)
		names = append(names, baseName)
		for _, fn := range out.Functions {
			names = append(names, fn.Name)
		}
		fmt.Fprintf(&reexports, "export { %s } from '../generated/%s.client';\n", strings.Join(names, ", "), baseName)
	}

	// Create the main client class
	var clientClass strings.Builder
	clientClass.WriteString(`export interface ForstClientConfig {
  baseUrl?: string;
  timeout?: number;
  retries?: number;
}

export class ForstClient {
`)

	// Add package properties
	clientClass.WriteString(strings.Join(packageProperties, "\n"))
	clientClass.WriteString("\n\n  constructor(config?: ForstClientConfig) {\n")
	clientClass.WriteString("    const client = createInvokeClient(config);\n")

	// Initialize package properties
	for _, export := range exports {
		fmt.Fprintf(&clientClass, "    this.%s = %s(client);\n", export, export)
	}

	clientClass.WriteString("  }\n}\n")

	// Combine all parts
	content := "// Auto-generated Forst Client\n"
	content += "// Generated by Forst TypeScript Transformer\n"
	content += "// Embedded invoke: set FORST_BASE_URL=http://127.0.0.1:8081\n\n"
	content += "import { createInvokeClient } from '@forst/client';\n"

	if len(imports) > 0 {
		content += strings.Join(imports, "\n") + "\n\n"
	}

	content += clientClass.String() + "\n"
	if reexports.Len() > 0 {
		content += "\n" + reexports.String()
	}
	content += "export type * from './types';\n"

	return content
}

// generateClientPackageJSON creates the package.json for the client
func generateClientPackageJSON() string {
	return `{
  "name": "@forst/generated-client",
  "private": true,
  "version": "0.1.0",
  "description": "Auto-generated Forst client",
  "sideEffects": true,
  "main": "index.ts",
  "types": "index.ts",
  "dependencies": {
    "@forst/client": "^0.1.0"
  },
  "devDependencies": {
    "typescript": "^5.0.0"
  }
}`
}

// copyFile copies a file from source to destination
func copyFile(src, dst string) error {
	input, err := generateIO.ReadFile(src)
	if err != nil {
		return err
	}

	err = generateIO.WriteFile(dst, input, 0644)
	if err != nil {
		return err
	}

	return nil
}
