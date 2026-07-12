package compiler

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/discovery"
	"forst/internal/ftconfig"
	"forst/internal/forstpkg"
	"forst/internal/generators"
	"forst/internal/modulecheck"
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

// LoadAndParse loads compile input and parses to AST nodes.
func (c *Compiler) LoadAndParse() ([]ast.Node, error) {
	return c.loadInputNodesForCompile()
}

// Typecheck runs module-aware typechecking on parsed nodes.
func (c *Compiler) Typecheck(nodes []ast.Node) (*typechecker.TypeChecker, *modulecheck.ModuleResult, error) {
	return c.typecheckForCompile(nodes)
}

// Transform generates Go source from a typechecked AST.
func (c *Compiler) Transform(checker *typechecker.TypeChecker, nodes []ast.Node) (string, error) {
	out, err := c.transformCheckedNodes(checker, nil, nodes)
	if err != nil {
		return "", err
	}
	return out.Main, nil
}

// CompileWithNodeRuntime compiles a Forst file and returns main and optional companion Go sources.
func (c *Compiler) CompileWithNodeRuntime() (main string, nodeRuntime string, invokeServer string, extraPackages map[string]string, extraImports map[string]string, err error) {
	out, err := c.compileToGo()
	if err != nil {
		return "", "", "", nil, nil, err
	}
	return out.Main, out.NodeRuntime, out.InvokeServer, out.ExtraPackages, out.ExtraPackageImports, nil
}

type compileGoOutput struct {
	Main          string
	NodeRuntime   string
	InvokeServer  string
	ExtraPackages       map[string]string // forst package name -> Go source
	ExtraPackageImports map[string]string // forst package name -> Go import path
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

	out, err := c.transformCheckedNodes(checker, modResult, forstNodes)
	if err != nil {
		return compileGoOutput{}, err
	}

	memAfter = getMemStats()
	c.logMemUsage("code generation", memBefore, memAfter)

	if c.Args.OutputPath != "" {
		if err := os.WriteFile(c.Args.OutputPath, []byte(out.Main), 0644); err != nil {
			return compileGoOutput{}, fmt.Errorf("error writing output file: %v", err)
		}
		if out.NodeRuntime != "" {
			runtimePath := nodeRuntimeOutputPath(c.Args.OutputPath)
			if err := os.WriteFile(runtimePath, []byte(out.NodeRuntime), 0644); err != nil {
				return compileGoOutput{}, fmt.Errorf("error writing node runtime file: %v", err)
			}
		}
		if out.InvokeServer != "" {
			invokePath := invokeServerOutputPath(c.Args.OutputPath)
			if err := os.WriteFile(invokePath, []byte(out.InvokeServer), 0644); err != nil {
				return compileGoOutput{}, fmt.Errorf("error writing invoke server file: %v", err)
			}
		}
	} else if c.Args.LogLevel == "trace" {
		c.log.Info("Generated Go code:")
		fmt.Println(out.Main)
		if out.NodeRuntime != "" {
			c.log.Info("Generated node runtime Go code:")
			fmt.Println(out.NodeRuntime)
		}
		if out.InvokeServer != "" {
			c.log.Info("Generated invoke server Go code:")
			fmt.Println(out.InvokeServer)
		}
	}

	return out, nil
}

func (c *Compiler) transformCheckedNodes(checker *typechecker.TypeChecker, modResult *modulecheck.ModuleResult, forstNodes []ast.Node) (compileGoOutput, error) {
	transformer := transformer_go.New(checker, c.log, c.Args.ExportStructFields)
	transformer.EmbedInvokeServer = c.embedInvokeEnabled()
	transformer.EmbedNodeHostMode = c.nodeHostModeEnabled()
	if c.Args.PackageRoot != "" {
		transformer.SandboxModulePath = "forst.run.temp"
	}
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

	if invokeServerCode == "" && c.embedInvokeEnabled() {
		if diag := c.embeddedInvokeMisconfigDiagnostic(transformer, forstNodes); diag != "" {
			return compileGoOutput{}, fmt.Errorf("%s", diag)
		}
	}

	needsNodeHostShutdown := invokeServerCode == "" && nodeRuntimeCode != "" && c.nodeHostModeEnabled()
	if invokeServerCode != "" {
		transformer.AppendInvokeShutdownIfNeeded()
	} else if needsNodeHostShutdown {
		transformer.AppendNodeHostShutdownIfNeeded()
	}
	if invokeServerCode != "" || needsNodeHostShutdown {
		goAST, err = transformer.Output.GenerateFile()
		if err != nil {
			return compileGoOutput{}, err
		}
		goCode, err = generateGoCodeCompile(goAST)
		if err != nil {
			return compileGoOutput{}, err
		}
	}

	extraPkgs, extraImports, err := c.compileExtraInvokePackages(modResult, forstNodes, invokeServerCode != "")
	if err != nil {
		return compileGoOutput{}, err
	}

	return compileGoOutput{
		Main: goCode, NodeRuntime: nodeRuntimeCode, InvokeServer: invokeServerCode,
		ExtraPackages: extraPkgs, ExtraPackageImports: extraImports,
	}, nil
}

func canonicalForstPackageImportPath(modResult *modulecheck.ModuleResult, forstPkg string) string {
	if modResult == nil || forstPkg == "" {
		return ""
	}
	var candidates []string
	for imp, pkg := range modResult.ImportPathToForstPkg() {
		if pkg == forstPkg {
			candidates = append(candidates, imp)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	best := candidates[0]
	for _, c := range candidates[1:] {
		if len(c) > len(best) {
			best = c
		}
	}
	return best
}

func (c *Compiler) compileExtraInvokePackages(modResult *modulecheck.ModuleResult, entryNodes []ast.Node, hasCompanion bool) (map[string]string, map[string]string, error) {
	if !hasCompanion || modResult == nil || c.Args.PackageRoot == "" {
		return nil, nil, nil
	}
	entryPkg := forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(entryNodes))
	needed := make(map[string]struct{})
	moduleFns, err := discovery.CollectInvokeFunctionsFromModule(c.log, RunBoundaryRoot(c.Args))
	if err != nil {
		return nil, nil, err
	}
	for _, fn := range moduleFns {
		if fn.Package != entryPkg && fn.Package != "" {
			needed[fn.Package] = struct{}{}
		}
	}
	if len(needed) == 0 {
		return nil, nil, nil
	}
	out := make(map[string]string)
	imports := make(map[string]string)
	for pkg := range needed {
		nodes := modResult.PerPackageNodes[pkg]
		tc := modResult.PerPackage[pkg]
		if nodes == nil || tc == nil {
			continue
		}
		tr := transformer_go.New(tc, c.log, c.Args.ExportStructFields)
		tr.SetModuleResult(modResult)
		goAST, err := transformForstFileToGoCompile(tr, nodes)
		if err != nil {
			return nil, nil, fmt.Errorf("extra package %q: %w", pkg, err)
		}
		code, err := generateGoCodeCompile(goAST)
		if err != nil {
			return nil, nil, fmt.Errorf("extra package %q emit: %w", pkg, err)
		}
		out[pkg] = code
		imports[pkg] = canonicalForstPackageImportPath(modResult, pkg)
	}
	return out, imports, nil
}

func (c *Compiler) embeddedInvokeMisconfigDiagnostic(transformer *transformer_go.Transformer, entryNodes []ast.Node) string {
	if transformer == nil || !transformer.EmbedInvokeServer || !transformer.IsMainPackage() {
		return ""
	}
	if !entryNodesHaveFuncMain(entryNodes) {
		return ""
	}
	boundary := RunBoundaryRoot(c.Args)
	if boundary == "" {
		return ""
	}
	moduleFns, err := discovery.CollectInvokeFunctionsFromModule(c.log, boundary)
	if err != nil || len(moduleFns) == 0 {
		return ""
	}
	compiledPkg := transformer.Output.PackageName()
	if compiledPkg == "" {
		compiledPkg = "main"
	}
	cross := discovery.CrossPackageInvokeExports(moduleFns, compiledPkg)
	if len(cross) == 0 {
		return ""
	}
	var names []string
	for _, fn := range cross {
		names = append(names, fmt.Sprintf("%s.%s", fn.Package, fn.Name))
	}
	return fmt.Sprintf(
		"embedded invoke: runnable exports found in other packages (%s) but not in compiled package %q\n"+
			"  boundary: %s\n"+
			"  help: move invoke exports to package main, or upgrade to multi-package invoke support",
		strings.Join(names, ", "), compiledPkg, boundary,
	)
}

func entryNodesHaveFuncMain(nodes []ast.Node) bool {
	for _, node := range nodes {
		fn, ok := node.(ast.FunctionNode)
		if !ok {
			continue
		}
		if fn.Ident.ID == "main" {
			return true
		}
	}
	return false
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
	if transformer == nil || !c.embedInvokeEnabled() {
		return "", nil
	}
	boundary := RunBoundaryRoot(c.Args)
	if boundary != "" && c.Args.PackageRoot != "" {
		functions, err := discovery.CollectInvokeFunctionsFromModule(c.log, boundary)
		if err != nil {
			return "", fmt.Errorf("embedded invoke: discover exports: %w", err)
		}
		if len(functions) > 0 {
			return transformer.InvokeServerSourceFromFunctions(true, functions)
		}
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

func (c *Compiler) nodeHostModeEnabled() bool {
	root := RunBoundaryRoot(c.Args)
	if root == "" {
		return false
	}
	cfg, err := ftconfig.LoadFromDir(root)
	if err != nil {
		return false
	}
	return cfg.Node.HostMode
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
	Debug(args ...any)
}, checker *typechecker.TypeChecker) {
	line := FormatNodeRuntimeLogLine(checker)
	if checker == nil || !checker.NeedsNodeRuntime() {
		log.Debug(line)
		return
	}
	log.Info(line)
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
