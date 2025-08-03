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

// processForstFile processes a single Forst file and generates TypeScript declaration files and client implementation
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
	tc := typechecker.New(log, false)
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

	// Write types declaration file (shared across all packages)
	typesPath := filepath.Join(generatedDir, "types.d.ts")
	typesCode := output.GenerateTypesFile()
	if err := os.WriteFile(typesPath, []byte(typesCode), 0644); err != nil {
		return fmt.Errorf("failed to write types declaration file: %w", err)
	}
	log.Infof("Generated types declaration file: %s", typesPath)

	// Write package declaration file with same basename as original Forst file
	declarationPath := filepath.Join(generatedDir, baseName+".d.ts")
	declarationCode := output.GenerateClientFile()
	if err := os.WriteFile(declarationPath, []byte(declarationCode), 0644); err != nil {
		return fmt.Errorf("failed to write declaration file: %w", err)
	}
	log.Infof("Generated declaration file: %s", declarationPath)

	// Write package client implementation file
	clientPath := filepath.Join(generatedDir, baseName+".client.ts")
	clientCode := generateClientImplementation(baseName, output)
	if err := os.WriteFile(clientPath, []byte(clientCode), 0644); err != nil {
		return fmt.Errorf("failed to write client implementation file: %w", err)
	}
	log.Infof("Generated client implementation file: %s", clientPath)

	return nil
}

// generateClientImplementation creates the client implementation file
func generateClientImplementation(packageName string, output *transformerts.TypeScriptOutput) string {
	content := "// Auto-generated Forst Client Implementation\n"
	content += "// Generated by Forst TypeScript Transformer\n\n"
	content += "import { ForstSidecarClient } from '@forst/sidecar';\n\n"

	// Add type definitions directly in the implementation file
	if len(output.Types) > 0 {
		content += "// Type definitions\n"
		for _, tsType := range output.Types {
			content += tsType + "\n\n"
		}
	}

	// Generate the client function
	content += fmt.Sprintf("export function %s(client: ForstSidecarClient) {\n", packageName)
	content += "  return {\n"

	// Generate function implementations based on the discovered functions
	// The functions are stored in output.Functions as TypeScript function signatures
	for _, functionSig := range output.Functions {
		// Extract function name from the signature
		// Function signatures are typically in format: "functionName(param1: type1, param2: type2): returnType"
		// We need to extract just the function name and create an implementation
		funcName := functionSig.Name
		if funcName != "" {
			// Build parameter list
			paramsSig := make([]string, len(functionSig.Parameters))
			for i, param := range functionSig.Parameters {
				paramsSig[i] = fmt.Sprintf("%s: %s", param.Name, param.Type)
			}
			paramsSigStr := strings.Join(paramsSig, ", ")

			paramNames := make([]string, len(functionSig.Parameters))
			for i, param := range functionSig.Parameters {
				paramNames[i] = param.Name
			}
			paramNamesStr := strings.Join(paramNames, ", ")

			// Get return type from function signature
			returnType := functionSig.ReturnType
			if returnType == "" {
				returnType = "any" // Fallback
			}

			content += fmt.Sprintf("    %s: async (%s) => {\n", funcName, paramsSigStr)
			content += fmt.Sprintf("      return client.invoke<%s>('%s', [%s]);\n", returnType, funcName, paramNamesStr)

			content += "    },\n"
		}
	}

	content += "  };\n"
	content += "}\n"

	return content
}

// extractFunctionName extracts the function name from a TypeScript function signature
func extractFunctionName(signature string) string {
	// Remove "export function " prefix if present
	funcName := signature
	if strings.HasPrefix(funcName, "export function ") {
		funcName = funcName[len("export function "):]
	}

	// Look for function name pattern: "functionName("
	// This is a simple extraction - in practice, you might want more robust parsing
	for i := 0; i < len(funcName); i++ {
		if funcName[i] == '(' {
			return funcName[:i]
		}
	}
	return ""
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

	// Copy types declaration file to client directory
	typesSource := filepath.Join(outputDir, "generated", "types.d.ts")
	typesDest := filepath.Join(clientDir, "types.d.ts")
	if err := copyFile(typesSource, typesDest); err != nil {
		return fmt.Errorf("failed to copy types declaration file: %w", err)
	}
	log.Infof("Copied types declaration file to client directory")

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
