package testrunner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"forst/internal/forstpkg"
	"forst/internal/generators"
	"forst/internal/goload"
	"forst/internal/modulecheck"
	transformer_go "forst/internal/transformer/go"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// Options configures a forst test run.
type Options struct {
	ModuleRoot string
	Paths      []string
	GoTestArgs []string
	Log        *logrus.Logger
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
	moduleRoot, err := filepath.Abs(moduleRoot)
	if err != nil {
		return ExitError, err
	}
	moduleRoot = goload.FindModuleRoot(moduleRoot)

	pkgs, err := DiscoverPackages(moduleRoot, opts.Paths)
	if err != nil {
		return ExitError, err
	}

	modResult, modErr := modulecheck.CheckModuleProviders(opts.Log, modulecheck.Options{ModuleRoot: moduleRoot})
	if modErr != nil {
		return ExitFailure, fmt.Errorf("module providers: %w", modErr)
	}

	// Emit Go for dependency Forst packages (library .ft only, not test packages under run).
	if modResult != nil {
		testDirs := make(map[string]struct{}, len(pkgs))
		for _, p := range pkgs {
			testDirs[p.Dir] = struct{}{}
		}
		if err := emitDependencyPackages(moduleRoot, modResult, testDirs, opts.Log); err != nil {
			return ExitFailure, err
		}
	}

	var failed bool
	for _, pkg := range pkgs {
		code, err := runPackageTests(moduleRoot, pkg, modResult, opts.GoTestArgs, opts.Log)
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

func emitDependencyPackages(moduleRoot string, modResult *modulecheck.ModuleResult, skipDirs map[string]struct{}, log *logrus.Logger) error {
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
		code, err := emitPackageGo(moduleRoot, pkg, modResult, log)
		if err != nil {
			return fmt.Errorf("%s: %w", pkg.RelPath, err)
		}
		genPath := filepath.Join(dir, "z_forst_gen.go")
		if err := os.WriteFile(genPath, []byte(code), 0o644); err != nil {
			return fmt.Errorf("%s: write generated: %w", pkg.RelPath, err)
		}
	}
	return nil
}

func emitPackageGo(moduleRoot string, pkg PackageUnderTest, modResult *modulecheck.ModuleResult, log *logrus.Logger) (string, error) {
	merged, _, err := forstpkg.ParseAndMergePackage(log, pkg.FtPaths)
	if err != nil {
		return "", fmt.Errorf("parse: %w", err)
	}

	forstPkg := forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(merged))

	var tc *typechecker.TypeChecker
	if modResult != nil && modResult.PerPackage[forstPkg] != nil {
		tc = modResult.PerPackage[forstPkg]
	} else {
		tc = typechecker.New(log, false)
		tc.GoWorkspaceDir = moduleRoot
		if err := tc.CheckTypes(merged); err != nil {
			return "", fmt.Errorf("typecheck: %w", err)
		}
	}

	tr := transformer_go.New(tc, log, false)
	if modResult != nil {
		tr.SetModuleResult(modResult)
	}
	goAST, err := tr.TransformForstFileToGo(merged)
	if err != nil {
		return "", fmt.Errorf("transform: %w", err)
	}
	return generators.GenerateGoCode(goAST)
}

func runPackageTests(moduleRoot string, pkg PackageUnderTest, modResult *modulecheck.ModuleResult, goTestArgs []string, log *logrus.Logger) (ExitCode, error) {
	code, err := emitPackageGo(moduleRoot, pkg, modResult, log)
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
	if rel, err := filepath.Rel(modRoot, pkg.Dir); err == nil && rel != "." {
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
	rel, err := filepath.Rel(moduleRoot, dir)
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

func indexOf(ss []string, s string) int {
	for i, v := range ss {
		if v == s {
			return i
		}
	}
	return -1
}
