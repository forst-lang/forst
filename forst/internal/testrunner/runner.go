package testrunner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"forst/internal/ast"
	"forst/internal/ftconfig"
	"forst/internal/forstpkg"
	"forst/internal/generators"
	"forst/internal/goload"
	"forst/internal/modulecheck"
	transformer_go "forst/internal/transformer/go"
	"forst/internal/typechecker"

	goast "go/ast"

	"github.com/sirupsen/logrus"
)

var (
	filepathAbs  = filepath.Abs
	filepathRel  = filepath.Rel
	transformForstFileToGo = func(tr *transformer_go.Transformer, merged []ast.Node) (*goast.File, error) {
		return tr.TransformForstFileToGo(merged)
	}
	generateGoCodeFn = generators.GenerateGoCode
)

// Options configures a forst test run.
type Options struct {
	ModuleRoot         string
	Paths              []string
	GoTestArgs         []string
	Log                *logrus.Logger
	ExportStructFields bool
}

// EmitOptions configures Go code generation for a package emit.
type EmitOptions struct {
	ExportStructFields bool // json-tagged exported struct fields (ftconfig / CLI)
	TestOnly           bool // z_forst_gen.go exists: transform *_test.ft only, omit type defs
}

// Run discovers Forst tests, emits Go, and invokes go test.
func Run(opts Options) (ExitCode, error) {
	if opts.Log == nil {
		opts.Log = logrus.New()
	}
	moduleRoot := opts.ModuleRoot
	if moduleRoot == "" {
		moduleRoot = "."
	}
	moduleRoot, err := filepathAbs(moduleRoot)
	if err != nil {
		return ExitError, err
	}
	moduleRoot = goload.FindModuleRoot(moduleRoot)

	emit := EmitOptions{ExportStructFields: opts.ExportStructFields}
	if !emit.ExportStructFields {
		emit.ExportStructFields = ftconfig.ExportStructFieldsFromDir(moduleRoot)
	}

	pkgs, err := DiscoverPackages(moduleRoot, opts.Paths)
	if err != nil {
		return ExitError, err
	}

	modResult, modErr := modulecheck.CheckModuleProviders(opts.Log, modulecheck.Options{ModuleRoot: moduleRoot})
	if modErr != nil {
		return ExitFailure, fmt.Errorf("module providers: %w", modErr)
	}

	var emittedLibDirs map[string]struct{}
	// Emit Go for dependency Forst packages (library .ft only, not test packages under run).
	if modResult != nil {
		testDirs := make(map[string]struct{}, len(pkgs))
		for _, p := range pkgs {
			testDirs[p.Dir] = struct{}{}
		}
		var emitErr error
		emittedLibDirs, emitErr = emitDependencyPackages(moduleRoot, modResult, testDirs, emit, opts.Log)
		if emitErr != nil {
			return ExitFailure, emitErr
		}
	}

	var failed bool
	for _, pkg := range pkgs {
		pkgEmit := emit
		// Test-only when this run emitted a library shim, or a prior run left z_forst_gen.go on disk.
		_, emittedThisRun := emittedLibDirs[pkg.Dir]
		pkgEmit.TestOnly = emittedThisRun || hasGeneratedLibrary(pkg.Dir)
		code, err := runPackageTests(moduleRoot, pkg, modResult, pkgEmit, opts.GoTestArgs, opts.Log)
		if err != nil {
			return ExitError, err
		}
		if code != ExitSuccess {
			failed = true
		}
	}
	if failed {
		return ExitFailure, nil
	}
	return ExitSuccess, nil
}

func emitDependencyPackages(moduleRoot string, modResult *modulecheck.ModuleResult, skipDirs map[string]struct{}, opts EmitOptions, log *logrus.Logger) (map[string]struct{}, error) {
	libOpts := opts
	libOpts.TestOnly = false
	emitted := make(map[string]struct{})
	for _, paths := range modResult.ForstPkgToFiles {
		if len(paths) == 0 {
			continue
		}
		dir := filepath.Dir(paths[0])
		if _, skip := skipDirs[dir]; skip {
			continue
		}
		var libPaths []string
		for _, p := range paths {
			if !IsTestForstFile(p) {
				libPaths = append(libPaths, p)
			}
		}
		if len(libPaths) == 0 {
			continue
		}
		pkg := PackageUnderTest{
			Dir:     dir,
			RelPath: relPath(moduleRoot, dir),
			FtPaths: libPaths,
		}
		code, err := emitPackageGo(moduleRoot, pkg, modResult, libOpts, log)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", pkg.RelPath, err)
		}
		genPath := filepath.Join(dir, "z_forst_gen.go")
		if err := os.WriteFile(genPath, []byte(code), 0o644); err != nil {
			return nil, fmt.Errorf("%s: write generated: %w", pkg.RelPath, err)
		}
		emitted[dir] = struct{}{}
	}
	return emitted, nil
}

func hasGeneratedLibrary(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "z_forst_gen.go"))
	return err == nil
}

func emitPackageGo(moduleRoot string, pkg PackageUnderTest, modResult *modulecheck.ModuleResult, opts EmitOptions, log *logrus.Logger) (string, error) {
	merged, byPath, err := forstpkg.ParseAndMergePackage(log, pkg.FtPaths)
	if err != nil {
		return "", fmt.Errorf("parse: %w", err)
	}

	forstPkg := forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(merged))

	var tc *typechecker.TypeChecker
	fromModule := modResult != nil && modResult.PerPackage[forstPkg] != nil
	if fromModule {
		tc = modResult.PerPackage[forstPkg]
	} else {
		tc = typechecker.New(log, false)
		tc.GoWorkspaceDir = moduleRoot
	}
	var savedProviders map[ast.Identifier][]typechecker.ProviderSlot
	if fromModule {
		savedProviders = cloneFunctionProviders(tc.FunctionProviders)
	}
	if err := tc.CheckTypes(merged); err != nil {
		return "", fmt.Errorf("typecheck: %w", err)
	}
	if fromModule && savedProviders != nil {
		tc.SetFunctionProviders(savedProviders)
		tc.FunctionProviders = savedProviders
	}

	transformPaths := pkg.FtPaths
	if opts.TestOnly {
		if len(pkg.TestPaths) == 0 {
			return "", fmt.Errorf("no test paths")
		}
		transformPaths = pkg.TestPaths
	}
	transformNodes := merged
	if opts.TestOnly {
		transformNodes, err = forstpkg.MergePackageASTsFromPaths(byPath, transformPaths)
		if err != nil {
			return "", fmt.Errorf("merge transform nodes: %w", err)
		}
	}

	tr := transformer_go.New(tc, log, opts.ExportStructFields)
	if opts.TestOnly {
		tr.OmitPackageTypeDefs = true
	}
	if modResult != nil {
		tr.SetModuleResult(modResult)
	}
	goAST, err := transformForstFileToGo(tr, transformNodes)
	if err != nil {
		return "", fmt.Errorf("transform: %w", err)
	}
	return generateGoCodeFn(goAST)
}

func runPackageTests(moduleRoot string, pkg PackageUnderTest, modResult *modulecheck.ModuleResult, opts EmitOptions, goTestArgs []string, log *logrus.Logger) (ExitCode, error) {
	code, err := emitPackageGo(moduleRoot, pkg, modResult, opts, log)
	if err != nil {
		return ExitFailure, fmt.Errorf("%s: %w", pkg.RelPath, err)
	}
	return writeGeneratedTestAndRun(pkg, code, goTestArgs, log)
}

func writeGeneratedTestAndRun(pkg PackageUnderTest, goCode string, goTestArgs []string, _ *logrus.Logger) (ExitCode, error) {
	genPath := filepath.Join(pkg.Dir, generatedTestGoName)
	if err := os.WriteFile(genPath, []byte(goCode), 0o644); err != nil {
		return ExitFailure, fmt.Errorf("%s: write generated test: %w", pkg.RelPath, err)
	}

	modRoot, err := goload.ModuleRootWithGoMod(pkg.Dir)
	if err != nil {
		return ExitFailure, fmt.Errorf("%s: %w (forst test packages need a local go.mod)", pkg.RelPath, err)
	}
	importPath := "."
	if rel, err := filepathRel(modRoot, pkg.Dir); err == nil && rel != "." {
		importPath = "./" + filepath.ToSlash(rel)
	}
	args := []string{"test"}
	args = append(args, goTestArgs...)
	args = append(args, importPath)

	cmd := exec.Command("go", args...)
	cmd.Dir = modRoot
	cmd.Env = append(os.Environ(), "GOWORK=off")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() >= 0 {
			return ExitCode(exitErr.ExitCode()), nil
		}
		return ExitFailure, fmt.Errorf("%s: go test: %w", pkg.RelPath, err)
	}
	_ = os.Remove(genPath)
	return ExitSuccess, nil
}

func relPath(moduleRoot, dir string) string {
	rel, err := filepathRel(moduleRoot, dir)
	if err != nil {
		return dir
	}
	return filepath.ToSlash(rel)
}

// ParseCLIArgs splits forst test CLI args into package paths and go test flags.
func ParseCLIArgs(args []string) (paths []string, goTestArgs []string) {
	if idx := indexOf(args, "--"); idx >= 0 {
		goTestArgs = append(goTestArgs, args[idx+1:]...)
		args = args[:idx]
	}
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			goTestArgs = append(goTestArgs, a)
		} else {
			paths = append(paths, a)
		}
	}
	return paths, goTestArgs
}

func cloneFunctionProviders(src map[ast.Identifier][]typechecker.ProviderSlot) map[ast.Identifier][]typechecker.ProviderSlot {
	if len(src) == 0 {
		return nil
	}
	out := make(map[ast.Identifier][]typechecker.ProviderSlot, len(src))
	for k, v := range src {
		slots := make([]typechecker.ProviderSlot, len(v))
		copy(slots, v)
		out[k] = slots
	}
	return out
}

func indexOf(ss []string, s string) int {
	for i, v := range ss {
		if v == s {
			return i
		}
	}
	return -1
}
