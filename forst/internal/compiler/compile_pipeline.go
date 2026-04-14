package compiler

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/generators"
	transformer_go "forst/internal/transformer/go"
	"forst/internal/typechecker"
	"os"
)

// CompileFile compiles a Forst file and returns the Go code.
func (c *Compiler) CompileFile() (*string, error) {
	forstNodes, err := c.loadInputNodesForCompile()
	if err != nil {
		return nil, err
	}

	c.reportPhase("Performing semantic analysis...")
	memBefore := getMemStats()

	checker := typechecker.New(c.log, c.Args.ReportPhases)
	checker.GoWorkspaceDir = c.goWorkspaceDirForCheck()
	if err := checker.CheckTypes(forstNodes); err != nil {
		c.log.Error("Encountered error checking types: ", err)
		checker.DebugPrintCurrentScope()
		return nil, err
	}

	memAfter := getMemStats()
	c.logMemUsage("semantic analysis", memBefore, memAfter)

	if c.Args.LogLevel == "debug" || c.Args.LogLevel == "trace" {
		c.debugPrintTypeInfo(checker)
	}

	c.reportPhase("Performing code generation...")
	memBefore = getMemStats()

	transformer := transformer_go.New(checker, c.log, c.Args.ExportStructFields)
	goAST, err := transformer.TransformForstFileToGo(forstNodes)
	if err != nil {
		return nil, err
	}

	if c.Args.LogLevel == "debug" || c.Args.LogLevel == "trace" {
		c.debugPrintGoAST(goAST)
	}

	goCode, err := generators.GenerateGoCode(goAST)
	if err != nil {
		return nil, err
	}

	memAfter = getMemStats()
	c.logMemUsage("code generation", memBefore, memAfter)

	if c.Args.OutputPath != "" {
		if err := os.WriteFile(c.Args.OutputPath, []byte(goCode), 0644); err != nil {
			return nil, fmt.Errorf("error writing output file: %v", err)
		}
	} else if c.Args.LogLevel == "trace" {
		c.log.Info("Generated Go code:")
		fmt.Println(goCode)
	}

	return &goCode, nil
}

func (c *Compiler) loadInputNodesForCompile() ([]ast.Node, error) {
	if c.Args.PackageRoot != "" {
		c.reportPhase("Loading merged package (same-package .ft files under -root)...")
		return c.loadMergedPackageAST()
	}
	return c.lexParseEntryFile()
}
