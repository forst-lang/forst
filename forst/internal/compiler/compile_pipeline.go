package compiler

import (
	"context"
	"fmt"
	"forst/internal/ast"
	"forst/internal/codegen/layout"
	"forst/internal/discovery"
	"forst/internal/forstpkg"
	"forst/internal/generators"
	"forst/internal/goload"
	"forst/internal/modulecheck"
	transformer_go "forst/internal/transformer/go"
	"forst/internal/typechecker"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
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

// CompileWithBridgeRuntime compiles a Forst file and returns main and optional companion Go sources.
func (c *Compiler) CompileWithBridgeRuntime() (main string, bridgeRuntime string, invokeServer string, extraPackages map[string]string, extraImports map[string]string, err error) {
	out, err := c.compileToGo()
	if err != nil {
		return "", "", "", nil, nil, err
	}
	return out.Main, out.BridgeRuntime, out.InvokeServer, out.ExtraPackages, out.ExtraPackageImports, nil
}

type compileGoOutput struct {
	Main                string
	BridgeRuntime       string
	InvokeServer  string
	ExtraPackages       map[string]string // forst package name -> Go source
	ExtraPackageImports map[string]string // forst package name -> Go import path
}

func (c *Compiler) compileToGo() (compileGoOutput, error) {
	profile := c.reloadProfileEnabled()
	var timings CompilePhaseTimings

	loadStart := time.Now()
	forstNodes, err := c.loadInputNodesForCompile()
	if err != nil {
		return compileGoOutput{}, err
	}
	if profile {
		timings.LoadParseMs = elapsedMs(loadStart)
	}

	c.reportPhase("Performing semantic analysis...")
	memBefore := getMemStats()

	typecheckStart := time.Now()
	checker, modResult, err := c.typecheckForCompile(forstNodes)
	if profile {
		timings.TypecheckMs = elapsedMs(typecheckStart)
	}
	if err != nil {
		c.log.Error("Encountered error checking types: ", err)
		if checker != nil {
			checker.DebugPrintCurrentScope()
		}
		return compileGoOutput{}, err
	}

	if err := checkRequireNoBridge(c.Args, checker); err != nil {
		return compileGoOutput{}, err
	}
	logBridgeRuntimeRequirement(c.log, checker)

	memAfter := getMemStats()
	c.logMemUsage("semantic analysis", memBefore, memAfter)

	if c.Args.LogLevel == "debug" || c.Args.LogLevel == "trace" {
		c.debugPrintTypeInfo(checker)
	}

	c.reportPhase("Performing code generation...")
	memBefore = getMemStats()

	codegenStart := time.Now()
	out, err := c.transformCheckedNodes(checker, modResult, forstNodes)
	if profile {
		timings.CodegenMs = elapsedMs(codegenStart)
	}
	if err != nil {
		return compileGoOutput{}, err
	}

	memAfter = getMemStats()
	c.logMemUsage("code generation", memBefore, memAfter)

	if c.Args.OutputPath != "" && c.Args.Command != "build" {
		if err := os.WriteFile(c.Args.OutputPath, []byte(out.Main), 0644); err != nil {
			return compileGoOutput{}, fmt.Errorf("error writing output file: %v", err)
		}
		if out.BridgeRuntime != "" {
			runtimePath := bridgeRuntimeOutputPath(c.Args.OutputPath)
			if err := removeLegacyBridgeRuntimeCompanions(c.Args.OutputPath); err != nil {
				return compileGoOutput{}, err
			}
			if err := os.WriteFile(runtimePath, []byte(out.BridgeRuntime), 0644); err != nil {
				return compileGoOutput{}, fmt.Errorf("error writing bridge runtime file: %v", err)
			}
		}
		if out.InvokeServer != "" {
			invokePath := invokeServerOutputPath(c.Args.OutputPath)
			if err := removeLegacyCompanionFile(legacyInvokeServerOutputPath(c.Args.OutputPath)); err != nil {
				return compileGoOutput{}, err
			}
			if err := os.WriteFile(invokePath, []byte(out.InvokeServer), 0644); err != nil {
				return compileGoOutput{}, fmt.Errorf("error writing invoke server file: %v", err)
			}
		}
		if err := WriteExtraPackagesForOutput(c.Args.OutputPath, out.ExtraPackages); err != nil {
			return compileGoOutput{}, err
		}
	} else if c.Args.LogLevel == "trace" {
		c.log.Info("Generated Go code:")
		fmt.Println(out.Main)
		if out.BridgeRuntime != "" {
			c.log.Info("Generated bridge runtime Go code:")
			fmt.Println(out.BridgeRuntime)
		}
		if out.InvokeServer != "" {
			c.log.Info("Generated invoke server Go code:")
			fmt.Println(out.InvokeServer)
		}
	}

	if profile {
		c.lastCompileTimings = timings
	}

	return out, nil
}

func (c *Compiler) transformCheckedNodes(checker *typechecker.TypeChecker, modResult *modulecheck.ModuleResult, forstNodes []ast.Node) (compileGoOutput, error) {
	transformer := transformer_go.New(checker, c.log, c.Args.ExportStructFields)
	transformer.EmbedInvokeServer = c.useEmbeddedInvokeRuntime()
	transformer.EmbedBridgeHostMode = c.bridgeHostModeEnabled()
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

	bridgeRuntimeCode, err := c.generateBridgeRuntimeCode(transformer)
	if err != nil {
		return compileGoOutput{}, err
	}

	moduleInvokeFns, err := c.resolveModuleInvokeFunctions(modResult)
	if err != nil {
		return compileGoOutput{}, err
	}

	invokeServerCode, err := c.generateInvokeServerCode(transformer, forstNodes, moduleInvokeFns)
	if err != nil {
		return compileGoOutput{}, err
	}

	if invokeServerCode == "" && c.useEmbeddedInvokeRuntime() {
		if diag := c.embeddedInvokeMisconfigDiagnostic(transformer, forstNodes, moduleInvokeFns); diag != "" {
			return compileGoOutput{}, fmt.Errorf("%s", diag)
		}
	}

	needsBridgeHostShutdown := invokeServerCode == "" && bridgeRuntimeCode != "" && c.bridgeHostModeEnabled()
	if invokeServerCode != "" {
		transformer.AppendInvokeShutdownIfNeeded()
	} else if needsBridgeHostShutdown {
		transformer.AppendNodeHostShutdownIfNeeded()
	}
	if invokeServerCode != "" || needsBridgeHostShutdown {
		goAST, err = transformer.Output.GenerateFile()
		if err != nil {
			return compileGoOutput{}, err
		}
		goCode, err = generateGoCodeCompile(goAST)
		if err != nil {
			return compileGoOutput{}, err
		}
	}

	extraPkgs, extraImports, err := c.compileExtraInvokePackages(modResult, forstNodes, invokeServerCode != "", moduleInvokeFns)
	if err != nil {
		return compileGoOutput{}, err
	}

	return compileGoOutput{
		Main: goCode, BridgeRuntime: bridgeRuntimeCode, InvokeServer: invokeServerCode,
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

func (c *Compiler) resolveModuleInvokeFunctions(modResult *modulecheck.ModuleResult) ([]discovery.FunctionInfo, error) {
	if !c.useEmbeddedInvokeRuntime() || RunBoundaryRoot(c.Args) == "" || c.Args.PackageRoot == "" {
		return nil, nil
	}
	if modResult != nil {
		return discovery.CollectInvokeFunctionsFromModuleResult(modResult), nil
	}
	boundary := RunBoundaryRoot(c.Args)
	fns, err := discovery.CollectInvokeFunctionsFromModule(c.log, boundary)
	if err != nil {
		if hint := goload.MissingGoModuleSetupHint(boundary); hint != "" && !strings.Contains(err.Error(), ".forst-gomod") {
			return nil, fmt.Errorf("embedded invoke: discover exports: %w; %s", err, hint)
		}
		return nil, fmt.Errorf("embedded invoke: discover exports: %w", err)
	}
	return fns, nil
}

func (c *Compiler) compileExtraInvokePackages(modResult *modulecheck.ModuleResult, entryNodes []ast.Node, hasCompanion bool, moduleFns []discovery.FunctionInfo) (map[string]string, map[string]string, error) {
	if !hasCompanion || modResult == nil || c.Args.PackageRoot == "" {
		return nil, nil, nil
	}
	entryPkg := forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(entryNodes))
	needed := make(map[string]struct{})
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
	var mu sync.Mutex
	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(4)
	for pkg := range needed {
		pkg := pkg
		nodes := modResult.PerPackageNodes[pkg]
		tc := modResult.PerPackage[pkg]
		if nodes == nil || tc == nil {
			continue
		}
		g.Go(func() error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			tr := transformer_go.New(tc, c.log, c.Args.ExportStructFields)
			tr.SetModuleResult(modResult)
			goAST, err := transformForstFileToGoCompile(tr, nodes)
			if err != nil {
				return fmt.Errorf("extra package %q: %w", pkg, err)
			}
			code, err := generateGoCodeCompile(goAST)
			if err != nil {
				return fmt.Errorf("extra package %q emit: %w", pkg, err)
			}
			mu.Lock()
			out[pkg] = code
			imports[pkg] = canonicalForstPackageImportPath(modResult, pkg)
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}
	return out, imports, nil
}

func (c *Compiler) embeddedInvokeMisconfigDiagnostic(transformer *transformer_go.Transformer, entryNodes []ast.Node, moduleFns []discovery.FunctionInfo) string {
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
	if len(moduleFns) == 0 {
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
			"  help: rebuild local forst (task build) or set FORST_BINARY to a compiler with cross-package embedded invoke support",
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

func (c *Compiler) generateBridgeRuntimeCode(transformer *transformer_go.Transformer) (string, error) {
	if transformer == nil {
		return "", nil
	}
	runtimeAST, err := transformer.BridgeRuntimeFile()
	if err != nil {
		return "", err
	}
	if runtimeAST == nil {
		return "", nil
	}
	return generateGoCodeCompile(runtimeAST)
}

func bridgeRuntimeOutputPath(outputPath string) string {
	ext := filepath.Ext(outputPath)
	base := strings.TrimSuffix(outputPath, ext)
	if ext == "" {
		return base + "_forst_0_bridge_runtime.gen.go"
	}
	return base + "_forst_0_bridge_runtime.gen" + ext
}

func legacyBridgeRuntimeOutputPath(outputPath string) string {
	ext := filepath.Ext(outputPath)
	base := strings.TrimSuffix(outputPath, ext)
	if ext == "" {
		return base + "_forst_0_node_runtime.gen.go"
	}
	return base + "_forst_0_node_runtime.gen" + ext
}

func removeLegacyBridgeRuntimeCompanions(outputPath string) error {
	return removeLegacyCompanionFile(legacyBridgeRuntimeOutputPath(outputPath))
}

func invokeServerOutputPath(outputPath string) string {
	ext := filepath.Ext(outputPath)
	base := strings.TrimSuffix(outputPath, ext)
	if ext == "" {
		return base + "_forst_1_invoke_server.gen.go"
	}
	return base + "_forst_1_invoke_server.gen" + ext
}

func legacyInvokeServerOutputPath(outputPath string) string {
	ext := filepath.Ext(outputPath)
	base := strings.TrimSuffix(outputPath, ext)
	if ext == "" {
		return base + "_forst_invoke_server.gen.go"
	}
	return base + "_forst_invoke_server.gen" + ext
}

func removeLegacyCompanionFile(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove legacy companion %s: %w", path, err)
	}
	return nil
}

// WriteExtraPackagesForOutput writes cross-package invoke Go sources beside a -o main output path.
func WriteExtraPackagesForOutput(outputPath string, extraPackages map[string]string) error {
	if len(extraPackages) == 0 {
		return nil
	}
	outDir := filepath.Dir(outputPath)
	for pkg, code := range extraPackages {
		pkgDir := filepath.Join(outDir, pkg)
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			return fmt.Errorf("mkdir extra package %q: %w", pkg, err)
		}
		pkgPath := filepath.Join(pkgDir, pkg+layout.SuffixGen)
		if err := os.WriteFile(pkgPath, []byte(code), 0644); err != nil {
			return fmt.Errorf("write extra package %q: %w", pkg, err)
		}
	}
	return nil
}

func (c *Compiler) generateInvokeServerCode(transformer *transformer_go.Transformer, nodes []ast.Node, moduleFns []discovery.FunctionInfo) (string, error) {
	if transformer == nil || !c.useEmbeddedInvokeRuntime() {
		return "", nil
	}
	boundary := RunBoundaryRoot(c.Args)
	if boundary != "" && c.Args.PackageRoot != "" {
		if len(moduleFns) > 0 {
			return transformer.InvokeServerSourceFromFunctions(true, moduleFns)
		}
	}
	return transformer.InvokeServerSource(c.useEmbeddedInvokeRuntime(), nodes)
}

func (c *Compiler) embedInvokeEnabled() bool {
	cfg, err := c.loadFtconfig()
	if err != nil || cfg == nil {
		return false
	}
	return cfg.Server.Embedded
}

// useEmbeddedInvokeRuntime reports whether this compile should emit invoke companions
// and ForstInvokeWaitForShutdown. generate/--go-out always emits plain Go; invoke glue
// is only for run/build/dev when server.embedded is on.
func (c *Compiler) useEmbeddedInvokeRuntime() bool {
	if !c.embedInvokeEnabled() {
		return false
	}
	switch c.Args.Command {
	case "run", "build", "dev":
		return true
	default:
		return false
	}
}

func (c *Compiler) bridgeHostModeEnabled() bool {
	cfg, err := c.loadFtconfig()
	if err != nil || cfg == nil {
		return false
	}
	return cfg.Bridge.HostMode
}

// PreferPackageDirRunEmit sets OutputPath beside the entry only when `generate.go` is
// configured in ftconfig (or the caller already set -o). Otherwise OutputPath stays empty
// so `forst run` uses the temp sandbox and does not litter `*.gen.go` next to source.
// Embedded invoke / bridge host mode always keep the isolated sandbox.
func (c *Compiler) PreferPackageDirRunEmit() {
	if c.Args.Command != "run" || c.Args.OutputPath != "" {
		return
	}
	if c.useEmbeddedInvokeRuntime() || c.bridgeHostModeEnabled() {
		return
	}
	if out := c.configuredPackageGoOut(); out != "" {
		c.Args.OutputPath = out
	}
}

// configuredPackageGoOut returns generate.go.out when ftconfig configures Go emit.
func (c *Compiler) configuredPackageGoOut() string {
	boundary := RunBoundaryRoot(c.Args)
	if boundary == "" {
		return ""
	}
	cfg, err := c.loadFtconfig()
	if err != nil || cfg == nil || !cfg.Generate.Go.IsConfigured() {
		return ""
	}
	return cfg.Generate.Go.EffectiveGoOut(boundary)
}

// defaultPackageGoOut resolves the Go emit path for plain `forst build`: configured
// generate.go.out first, else stem.gen.go beside the entry (needed for `go build .`).
func (c *Compiler) defaultPackageGoOut() string {
	if out := c.configuredPackageGoOut(); out != "" {
		return out
	}
	entry := c.Args.FilePath
	if entry == "" {
		return ""
	}
	dir := filepath.Dir(entry)
	stem := strings.TrimSuffix(filepath.Base(entry), filepath.Ext(entry))
	if stem == "" {
		stem = "main"
	}
	return filepath.Join(dir, stem+".gen.go")
}

func checkRequireNoBridge(args Args, checker *typechecker.TypeChecker) error {
	if !args.RequireNoBridge {
		return nil
	}
	if checker != nil && checker.NeedsBridgeRuntime() {
		return fmt.Errorf("program requires Node runtime (opted-in TypeScript imports); cannot build with -require-no-bridge")
	}
	return nil
}

func logBridgeRuntimeRequirement(log interface {
	Info(args ...any)
	Debug(args ...any)
}, checker *typechecker.TypeChecker) {
	line := FormatBridgeRuntimeLogLine(checker)
	if checker == nil || !checker.NeedsBridgeRuntime() {
		log.Debug(line)
		return
	}
	log.Info(line)
}

// FormatBridgeRuntimeLogLine returns the post-typecheck node runtime summary for CLI output.
func FormatBridgeRuntimeLogLine(checker *typechecker.TypeChecker) string {
	if checker == nil || !checker.NeedsBridgeRuntime() {
		return "bridge runtime: not required"
	}
	modules, exports, moduleIDs := checker.BridgeRuntimeSummary()
	if len(moduleIDs) == 0 {
		return fmt.Sprintf("bridge runtime: required (%d modules, %d exports)", modules, exports)
	}
	return fmt.Sprintf("bridge runtime: required (%d modules, %d exports) — %s",
		modules, exports, strings.Join(moduleIDs, ", "))
}

func (c *Compiler) loadInputNodesForCompile() ([]ast.Node, error) {
	if c.Args.PackageRoot != "" {
		c.reportPhase("Loading merged package (same-package .ft files under -root)...")
		return c.loadMergedPackageAST()
	}
	return c.lexParseEntryFile()
}
