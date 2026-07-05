package compiler

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/generators"
	transformer_go "forst/internal/transformer/go"
	"os"

	goast "go/ast"
)

var (
	transformForstFileToGoCompile = func(tr *transformer_go.Transformer, nodes []ast.Node) (*goast.File, error) {
		return tr.TransformForstFileToGo(nodes)
	}
	generateGoCodeCompile = generators.GenerateGoCode
)

// CompileFile compiles a Forst file and returns the Go code.
func (c *Compiler) CompileFile() (*string, error) {
	forstNodes, err := c.loadInputNodesForCompile()
	if err != nil {
		return nil, err
	}

	c.reportPhase("Performing semantic analysis...")
	memBefore := getMemStats()

	checker, modResult, err := c.typecheckForCompile(forstNodes)
	if err != nil {
		c.log.Error("Encountered error checking types: ", err)
		if checker != nil {
			checker.DebugPrintCurrentScope()
		}
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
	if modResult != nil {
		transformer.SetModuleResult(modResult)
	}
	goAST, err := transformForstFileToGoCompile(transformer, forstNodes)
	if err != nil {
		return nil, err
	}

	if c.Args.LogLevel == "debug" || c.Args.LogLevel == "trace" {
		c.debugPrintGoAST(goAST)
	}

	goCode, err := generateGoCodeCompile(goAST)
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
