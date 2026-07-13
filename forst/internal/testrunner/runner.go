package testrunner

import (
	"fmt"
	"path/filepath"
	"strings"

	"forst/internal/ast"
	"forst/internal/ftconfig"
	"forst/internal/forstpkg"
	"forst/internal/goload"
	"forst/internal/modulecheck"
	"forst/internal/project"
	transformer_go "forst/internal/transformer/go"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// Options configures a forst test run.
type Options struct {
	ModuleRoot         string
	BoundaryRoot       string // ftconfig boundary (-root); defaults to ModuleRoot
	Paths              []string
	GoTestArgs         []string
	Log                *logrus.Logger
	ExportStructFields bool
}

// EmitOptions configures Go code generation for a package emit (unit tests).
type EmitOptions struct {
	ExportStructFields bool
	TestOnly           bool // emit *_test.ft nodes only, omit package type defs
}

// Run discovers Forst tests, emits Go under .forst/gen/test/, and invokes go test.
func Run(opts Options) (ExitCode, error) {
	if opts.Log == nil {
		opts.Log = logrus.New()
	}
	moduleRoot := opts.ModuleRoot
	if moduleRoot == "" {
		moduleRoot = "."
	}
	moduleRoot, err := currentFilepathAbs()(moduleRoot)
	if err != nil {
		return ExitError, err
	}
	moduleRoot = goload.FindModuleRoot(moduleRoot)
	if !opts.ExportStructFields {
		opts.ExportStructFields = ftconfig.ExportStructFieldsFromDir(moduleRoot)
	}

	proj, err := project.Open(opts.Log, project.OpenOpts{
		BoundaryRoot: boundaryRoot(opts),
		Cwd:          moduleRoot,
	})
	if err != nil {
		return ExitFailure, fmt.Errorf("project open: %w", err)
	}
	return RunWithProject(proj, opts)
}

func boundaryRoot(opts Options) string {
	if opts.BoundaryRoot != "" {
		return opts.BoundaryRoot
	}
	return opts.ModuleRoot
}

// emitPackageGo transforms a package for unit tests of the emit pipeline.
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

	if opts.TestOnly {
		tc.RebindScopes(merged, transformNodes)
	}

	tr := transformer_go.New(tc, log, opts.ExportStructFields)
	if opts.TestOnly {
		tr.OmitPackageTypeDefs = true
	}
	if modResult != nil {
		tr.SetModuleResult(modResult)
	}
	goAST, err := currentTransformForstFileToGo()(tr, transformNodes)
	if err != nil {
		return "", fmt.Errorf("transform: %w", err)
	}
	return currentGenerateGoCodeFn()(goAST)
}

func relPath(moduleRoot, dir string) string {
	rel, err := currentFilepathRel()(moduleRoot, dir)
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
