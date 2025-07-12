package main

import (
	"fmt"
	"forst/internal/lexer"
	"forst/internal/parser"
	transformerts "forst/internal/transformer/ts"
	"forst/internal/typechecker"
	"os"
	"path/filepath"
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

	// Process each Forst file
	for _, file := range forstFiles {
		if err := processForstFile(file, outputDir, log); err != nil {
			log.Errorf("Failed to process %s: %v", file, err)
			continue
		}
	}

	// Generate client package structure
	if err := generateClientPackage(outputDir, forstFiles, log); err != nil {
		log.Errorf("Failed to generate client package: %v", err)
	}

	log.Info("TypeScript client generation completed")
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

// processForstFile processes a single Forst file and generates TypeScript code
func processForstFile(filePath, targetDir string, log *logrus.Logger) error {
	log.Infof("Processing %s", filePath)

	// Read the Forst file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Lex the content into tokens
	lex := lexer.New(content, filePath, log)
	tokens := lex.Lex()

	log.Debugf("Lexed %d tokens", len(tokens))

	// Parse the tokens into AST
	p := parser.New(tokens, filePath, log)
	ast, err := p.ParseFile()
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	log.Debugf("Parsed %d AST nodes", len(ast))

	// Create typechecker and run type checking
	tc := typechecker.New(log, false) // reportPhases = false
	if err := tc.CheckTypes(ast); err != nil {
		return fmt.Errorf("failed to type check: %w", err)
	}

	log.Debug("Type checking completed")

	// Create the TypeScript transformer
	transformer := transformerts.New(tc, log)

	// Transform the AST to TypeScript
	output, err := transformer.TransformForstFileToTypeScript(ast)
	if err != nil {
		return fmt.Errorf("failed to transform to TypeScript: %w", err)
	}

	// Generate the output file paths
	fileName := filepath.Base(filePath)
	baseName := fileName[:len(fileName)-len(filepath.Ext(fileName))]

	// Create generated directory
	generatedDir := filepath.Join(targetDir, "generated")
	if err := os.MkdirAll(generatedDir, 0755); err != nil {
		return fmt.Errorf("failed to create generated directory: %w", err)
	}

	// Write types file (shared across all packages)
	typesPath := filepath.Join(generatedDir, "types.ts")
	typesCode := output.GenerateTypesFile()
	if err := os.WriteFile(typesPath, []byte(typesCode), 0644); err != nil {
		return fmt.Errorf("failed to write types file: %w", err)
	}
	log.Infof("Generated types file: %s", typesPath)

	// Write package client file
	clientPath := filepath.Join(generatedDir, baseName+".client.ts")
	clientCode := output.GenerateClientFile()
	if err := os.WriteFile(clientPath, []byte(clientCode), 0644); err != nil {
		return fmt.Errorf("failed to write client file: %w", err)
	}
	log.Infof("Generated client file: %s", clientPath)

	return nil
}

// generateClientPackage creates the main client package structure
func generateClientPackage(outputDir string, forstFiles []string, log *logrus.Logger) error {
	// Create client package directory
	clientDir := filepath.Join(outputDir, "client")
	if err := os.MkdirAll(clientDir, 0755); err != nil {
		return fmt.Errorf("failed to create client directory: %w", err)
	}

	// Generate main client index file
	indexContent := generateClientIndex(forstFiles)
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

	// Copy types file to client directory
	typesSource := filepath.Join(outputDir, "generated", "types.ts")
	typesDest := filepath.Join(clientDir, "types.ts")
	if err := copyFile(typesSource, typesDest); err != nil {
		return fmt.Errorf("failed to copy types file: %w", err)
	}
	log.Infof("Copied types file to client directory")

	return nil
}

// generateClientIndex creates the main client index file
func generateClientIndex(forstFiles []string) string {
	var imports []string
	var exports []string
	var packageProperties []string

	// Generate imports and exports for each package
	for _, file := range forstFiles {
		fileName := filepath.Base(file)
		baseName := fileName[:len(fileName)-len(filepath.Ext(fileName))]

		// Import the client implementation
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
	content += "import { ForstClient as SidecarClient } from '@forst/sidecar';\n"

	if len(imports) > 0 {
		content += strings.Join(imports, "\n") + "\n\n"
	}

	content += clientClass + "\n"

	if len(exports) > 0 {
		content += "// Export individual packages\n"
		content += "export { " + strings.Join(exports, ", ") + " };\n\n"
	}

	content += "// Export types\n"
	content += "export * from './types';\n\n"
	content += "// Export default\n"
	content += "export default ForstClient;\n"

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
