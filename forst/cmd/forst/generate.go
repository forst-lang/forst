package main

import (
	"fmt"
	transformerts "forst/internal/transformer/ts"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

// generateCommand handles the "forst generate" command
func generateCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("generate command requires a target file or directory")
	}

	target := args[0]

	// Create logger
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)

	// Check if target is a file or directory
	fileInfo, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("failed to stat target %s: %w", target, err)
	}

	var forstFiles []string
	var outputDir string

	if fileInfo.IsDir() {
		// Target is a directory, find all .ft files
		forstFiles, err = findForstFiles(target)
		if err != nil {
			return fmt.Errorf("failed to find Forst files: %w", err)
		}
		outputDir = target
	} else {
		// Target is a single file
		if filepath.Ext(target) != ".ft" {
			return fmt.Errorf("target file must have .ft extension")
		}
		forstFiles = []string{target}
		outputDir = filepath.Dir(target)
	}

	if len(forstFiles) == 0 {
		log.Warn("No .ft files found in target directory")
		return nil
	}

	log.Infof("Found %d Forst files", len(forstFiles))

	sort.Strings(forstFiles)

	var outputs []*transformerts.TypeScriptOutput
	for _, file := range forstFiles {
		out, err := transformerts.TransformForstFileFromPath(file, log, transformerts.TransformForstFileOptions{
			RelaxedTypecheck: false,
		})
		if err != nil {
			log.Errorf("Failed to process %s: %v", file, err)
			continue
		}
		outputs = append(outputs, out)
	}

	if len(outputs) == 0 {
		log.Warn("No Forst files were successfully processed")
		return nil
	}

	merged, err := transformerts.MergeTypeScriptOutputs(outputs)
	if err != nil {
		return fmt.Errorf("merge TypeScript outputs: %w", err)
	}

	generatedDir := filepath.Join(outputDir, "generated")
	if err := os.MkdirAll(generatedDir, 0755); err != nil {
		return fmt.Errorf("failed to create generated directory: %w", err)
	}

	typesPath := filepath.Join(generatedDir, "types.d.ts")
	typesCode := merged.GenerateTypesFile()
	if err := os.WriteFile(typesPath, []byte(typesCode), 0644); err != nil {
		return fmt.Errorf("failed to write types declaration file: %w", err)
	}
	log.Infof("Generated types declaration file: %s", typesPath)

	for _, out := range outputs {
		stem := out.SourceFileStem
		clientPath := filepath.Join(generatedDir, stem+".client.ts")
		clientCode := out.GenerateClientFile()
		if err := os.WriteFile(clientPath, []byte(clientCode), 0644); err != nil {
			log.Errorf("Failed to write client module %s: %v", clientPath, err)
			continue
		}
		log.Infof("Generated client module: %s", clientPath)
	}

	stems := make([]string, len(outputs))
	for i, o := range outputs {
		stems[i] = o.SourceFileStem
	}
	sort.Strings(stems)

	// Generate client package structure (only entries that produced output)
	if err := generateClientPackage(outputDir, stems, log); err != nil {
		log.Errorf("Failed to generate client package: %v", err)
	}

	log.Info("TypeScript declaration files and client implementations generation completed")
	return nil
}

// findForstFiles finds all .ft files in the target directory
func findForstFiles(targetDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".ft" {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// generateClientPackage creates the main client package structure
func generateClientPackage(outputDir string, clientStems []string, log *logrus.Logger) error {
	// Create client package directory
	clientDir := filepath.Join(outputDir, "client")
	if err := os.MkdirAll(clientDir, 0755); err != nil {
		return fmt.Errorf("failed to create client directory: %w", err)
	}

	// Generate main client index file
	indexContent := generateClientIndex(clientStems)
	indexPath := filepath.Join(clientDir, "index.ts")
	if err := os.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
		return fmt.Errorf("failed to write client index: %w", err)
	}
	log.Infof("Generated client index: %s", indexPath)

	// Generate package.json for the client
	packageContent := generateClientPackageJson()
	packagePath := filepath.Join(clientDir, "package.json")
	if err := os.WriteFile(packagePath, []byte(packageContent), 0644); err != nil {
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

	return nil
}

// generateClientIndex creates the main client index file.
// clientStems are basenames without .ft (e.g. "api" for api.ft), sorted for stable output.
func generateClientIndex(clientStems []string) string {
	var imports []string
	var exports []string
	var packageProperties []string

	for _, baseName := range clientStems {
		imports = append(imports, fmt.Sprintf("import { %s } from '../generated/%s.client';", baseName, baseName))
		exports = append(exports, baseName)
		packageProperties = append(packageProperties, fmt.Sprintf("  public %s: ReturnType<typeof %s>;", baseName, baseName))
	}

	// Create the main client class
	clientClass := `export interface ForstClientConfig {
  baseUrl?: string;
  timeout?: number;
  retries?: number;
}

export class ForstClient {
  private client: SidecarClient;
`

	// Add package properties
	clientClass += strings.Join(packageProperties, "\n")
	clientClass += "\n\n  constructor(config?: ForstClientConfig) {\n"
	clientClass += "    const defaultConfig = {\n"
	clientClass += "      baseUrl: process.env.FORST_BASE_URL || 'http://localhost:8080',\n"
	clientClass += "      timeout: 30000,\n"
	clientClass += "      retries: 3,\n"
	clientClass += "      ...config,\n"
	clientClass += "    };\n\n"
	clientClass += "    this.client = new SidecarClient(defaultConfig);\n"

	// Initialize package properties
	for _, export := range exports {
		clientClass += fmt.Sprintf("    this.%s = %s(this.client);\n", export, export)
	}

	clientClass += "  }\n}\n"

	// Combine all parts
	content := "// Auto-generated Forst Client\n"
	content += "// Generated by Forst TypeScript Transformer\n\n"
	content += "import { ForstSidecarClient as SidecarClient } from '@forst/sidecar';\n"

	if len(imports) > 0 {
		content += strings.Join(imports, "\n") + "\n\n"
	}

	content += clientClass + "\n"

	return content
}

// generateClientPackageJson creates the package.json for the client
func generateClientPackageJson() string {
	return `{
  "name": "@forst/client",
  "version": "0.1.0",
  "description": "Auto-generated Forst client",
  "main": "index.ts",
  "types": "index.ts",
  "dependencies": {
    "@forst/sidecar": "^0.1.0"
  },
  "devDependencies": {
    "typescript": "^5.0.0"
  }
}`
}

// copyFile copies a file from source to destination
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		return err
	}

	return nil
}
