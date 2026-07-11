package compiler

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/ftconfig"
	"forst/internal/generators"
	transformer_go "forst/internal/transformer/go"
	"forst/internal/typechecker"
	"os"
	"path/filepath"
	"strings"

	goast "go/ast"
)

var (
	transformForstFileToGoCompile = func(tr *transformer_go.Transformer, nodes []ast.Node) (*goast.File, error) {
		return tr.TransformForstFileToGo(nodes)
	}
	generateGoCodeCompile = generators.GenerateGoCode
)

// CompileFile compiles a Forst file and returns the main Go code.
func (c *Compiler) CompileFile() (*string, error) {
	out, err := c.compileToGo()
	if err != nil {
		return nil, err
	}
	return &out.Main, nil
}

// CompileWithNodeRuntime compiles a Forst file and returns main and optional companion Go sources.
func (c *Compiler) CompileWithNodeRuntime() (main string, nodeRuntime string, invokeServer string, err error) {
	out, err := c.compileToGo()
	if err != nil {
		return "", "", "", err
	}
	return out.Main, out.NodeRuntime, out.InvokeServer, nil
}

type compileGoOutput struct {
	Main         string
	NodeRuntime  string
	InvokeServer string
}

func (c *Compiler) compileToGo() (compileGoOutput, error) {
	forstNodes, err := c.loadInputNodesForCompile()
	if err != nil {
		return compileGoOutput{}, err
	}

	c.reportPhase("Performing semantic analysis...")
	memBefore := getMemStats()

	checker, modResult, err := c.typecheckForCompile(forstNodes)
	if err != nil {
		c.log.Error("Encountered error checking types: ", err)
		if checker != nil {
			checker.DebugPrintCurrentScope()
		}
		return compileGoOutput{}, err
	}

	if err := checkRequireNoNode(c.Args, checker); err != nil {
		return compileGoOutput{}, err
	}
	logNodeRuntimeRequirement(c.log, checker)

	memAfter := getMemStats()
	c.logMemUsage("semantic analysis", memBefore, memAfter)

	if c.Args.LogLevel == "debug" || c.Args.LogLevel == "trace" {
		c.debugPrintTypeInfo(checker)
	}

	c.reportPhase("Performing code generation...")
	memBefore = getMemStats()

	transformer := transformer_go.New(checker, c.log, c.Args.ExportStructFields)
	transformer.EmbedInvokeServer = c.embedInvokeEnabled()
	if modResult != nil {
		transformer.SetModuleResult(modResult)
	}
	goAST, err := transformForstFileToGoCompile(transformer, forstNodes)
	if err != nil {
		return compileGoOutput{}, err
	}

	if c.Args.LogLevel == "debug" || c.Args.LogLevel == "trace" {
		c.debugPrintGoAST(goAST)
	}

	goCode, err := generateGoCodeCompile(goAST)
	if err != nil {
		return compileGoOutput{}, err
	}

	nodeRuntimeCode, err := c.generateNodeRuntimeCode(transformer)
	if err != nil {
		return compileGoOutput{}, err
	}

	invokeServerCode, err := c.generateInvokeServerCode(transformer, forstNodes)
	if err != nil {
		return compileGoOutput{}, err
	}

	memAfter = getMemStats()
	c.logMemUsage("code generation", memBefore, memAfter)

	if c.Args.OutputPath != "" {
		if err := os.WriteFile(c.Args.OutputPath, []byte(goCode), 0644); err != nil {
			return compileGoOutput{}, fmt.Errorf("error writing output file: %v", err)
		}
		if nodeRuntimeCode != "" {
			runtimePath := nodeRuntimeOutputPath(c.Args.OutputPath)
			if err := os.WriteFile(runtimePath, []byte(nodeRuntimeCode), 0644); err != nil {
				return compileGoOutput{}, fmt.Errorf("error writing node runtime file: %v", err)
			}
		}
		if invokeServerCode != "" {
			invokePath := invokeServerOutputPath(c.Args.OutputPath)
			if err := os.WriteFile(invokePath, []byte(invokeServerCode), 0644); err != nil {
				return compileGoOutput{}, fmt.Errorf("error writing invoke server file: %v", err)
			}
		}
	} else if c.Args.LogLevel == "trace" {
		c.log.Info("Generated Go code:")
		fmt.Println(goCode)
		if nodeRuntimeCode != "" {
			c.log.Info("Generated node runtime Go code:")
			fmt.Println(nodeRuntimeCode)
		}
		if invokeServerCode != "" {
			c.log.Info("Generated invoke server Go code:")
			fmt.Println(invokeServerCode)
		}
	}

	return compileGoOutput{Main: goCode, NodeRuntime: nodeRuntimeCode, InvokeServer: invokeServerCode}, nil
}

func (c *Compiler) generateNodeRuntimeCode(transformer *transformer_go.Transformer) (string, error) {
	if transformer == nil {
		return "", nil
	}
	runtimeAST, err := transformer.NodeRuntimeFile()
	if err != nil {
		return "", err
	}
	if runtimeAST == nil {
		return "", nil
	}
	return generateGoCodeCompile(runtimeAST)
}

func nodeRuntimeOutputPath(outputPath string) string {
	ext := filepath.Ext(outputPath)
	base := strings.TrimSuffix(outputPath, ext)
	if ext == "" {
		return base + "_forst_node_runtime.gen.go"
	}
	return base + "_forst_node_runtime.gen" + ext
}

func invokeServerOutputPath(outputPath string) string {
	ext := filepath.Ext(outputPath)
	base := strings.TrimSuffix(outputPath, ext)
	if ext == "" {
		return base + "_forst_invoke_server.gen.go"
	}
	return base + "_forst_invoke_server.gen" + ext
}

func (c *Compiler) generateInvokeServerCode(transformer *transformer_go.Transformer, nodes []ast.Node) (string, error) {
	if transformer == nil {
		return "", nil
	}
	return transformer.InvokeServerSource(c.embedInvokeEnabled(), nodes)
}

func (c *Compiler) embedInvokeEnabled() bool {
	root := RunBoundaryRoot(c.Args)
	if root == "" {
		return false
	}
	cfg, err := ftconfig.LoadFromDir(root)
	if err != nil {
		return false
	}
	return cfg.Server.Embedded
}

func checkRequireNoNode(args Args, checker *typechecker.TypeChecker) error {
	if !args.RequireNoNode {
		return nil
	}
	if checker != nil && checker.NeedsNodeRuntime() {
		return fmt.Errorf("program requires Node runtime (opted-in TypeScript imports); cannot build with -require-no-node")
	}
	return nil
}

func logNodeRuntimeRequirement(log interface {
	Info(args ...any)
}, checker *typechecker.TypeChecker) {
	log.Info(FormatNodeRuntimeLogLine(checker))
}

// FormatNodeRuntimeLogLine returns the post-typecheck node runtime summary for CLI output.
func FormatNodeRuntimeLogLine(checker *typechecker.TypeChecker) string {
	if checker == nil || !checker.NeedsNodeRuntime() {
		return "node runtime: not required"
	}
	modules, exports, moduleIDs := checker.NodeRuntimeSummary()
	if len(moduleIDs) == 0 {
		return fmt.Sprintf("node runtime: required (%d modules, %d exports)", modules, exports)
	}
	return fmt.Sprintf("node runtime: required (%d modules, %d exports) — %s",
		modules, exports, strings.Join(moduleIDs, ", "))
}

func (c *Compiler) loadInputNodesForCompile() ([]ast.Node, error) {
	if c.Args.PackageRoot != "" {
		c.reportPhase("Loading merged package (same-package .ft files under -root)...")
		return c.loadMergedPackageAST()
	}
	return c.lexParseEntryFile()
}
