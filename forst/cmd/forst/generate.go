package main

import (
	"fmt"
	"forst/internal/lexer"
	"forst/internal/parser"
	transformerts "forst/internal/transformer/ts"
	"forst/internal/typechecker"
	"os"
	"path/filepath"

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

	// Generate the output file path
	fileName := filepath.Base(filePath)
	baseName := fileName[:len(fileName)-len(filepath.Ext(fileName))]
	outputPath := filepath.Join(targetDir, "generated", baseName+".d.ts")

	// Ensure the output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write the generated TypeScript declaration code
	generatedCode := output.GenerateFile()
	if err := os.WriteFile(outputPath, []byte(generatedCode), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	log.Infof("Generated %s", outputPath)
	return nil
}
