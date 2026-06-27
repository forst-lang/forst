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

// Run discovers Forst tests, emits Go, and invokes go test. Returns go test exit code.
func Run(opts Options) (int, error) {
	if opts.Log == nil {
		opts.Log = logrus.New()
	}
	moduleRoot := opts.ModuleRoot
	if moduleRoot == "" {
		moduleRoot = "."
	}
	moduleRoot, err := filepath.Abs(moduleRoot)
	if err != nil {
		return 2, err
	}
	moduleRoot = goload.FindModuleRoot(moduleRoot)

	pkgs, err := DiscoverPackages(moduleRoot, opts.Paths)
	if err != nil {
		return 2, err
	}

	var failed bool
	for _, pkg := range pkgs {
		code, err := runPackageTests(moduleRoot, pkg, opts.GoTestArgs, opts.Log)
		if err != nil {
			return 2, err
		}
		if code != 0 {
			failed = true
		}
	}
	if failed {
		return 1, nil
	}
	return 0, nil
}

func emitPackageGo(moduleRoot string, pkg PackageUnderTest, log *logrus.Logger) (string, error) {
	merged, _, err := forstpkg.ParseAndMergePackage(log, pkg.FtPaths)
	if err != nil {
		return "", fmt.Errorf("parse: %w", err)
	}
	tc := typechecker.New(log, false)
	tc.GoWorkspaceDir = moduleRoot
	if err := tc.CheckTypes(merged); err != nil {
		return "", fmt.Errorf("typecheck: %w", err)
	}
	tr := transformer_go.New(tc, log, false)
	goAST, err := tr.TransformForstFileToGo(merged)
	if err != nil {
		return "", fmt.Errorf("transform: %w", err)
	}
	return generators.GenerateGoCode(goAST)
}

func runPackageTests(moduleRoot string, pkg PackageUnderTest, goTestArgs []string, log *logrus.Logger) (int, error) {
	code, err := emitPackageGo(moduleRoot, pkg, log)
	if err != nil {
		return 1, fmt.Errorf("%s: %w", pkg.RelPath, err)
	}
	return writeGeneratedTestAndRun(pkg, code, goTestArgs, log)
}

func writeGeneratedTestAndRun(pkg PackageUnderTest, goCode string, goTestArgs []string, log *logrus.Logger) (int, error) {
	genPath := filepath.Join(pkg.Dir, generatedTestGoName)
	if err := os.WriteFile(genPath, []byte(goCode), 0o644); err != nil {
		return 1, fmt.Errorf("%s: write generated test: %w", pkg.RelPath, err)
	}
	defer func() {
		if err := os.Remove(genPath); err != nil && log != nil {
			log.Warnf("remove generated test file %s: %v", genPath, err)
		}
	}()

	// Resolve the module from the package dir (where we wrote the test file), not only the
	// caller's ModuleRoot — avoids go test running outside a module when paths disagree.
	modRoot := goload.FindModuleRoot(pkg.Dir)
	if _, err := os.Stat(filepath.Join(modRoot, "go.mod")); err != nil {
		return 1, fmt.Errorf("%s: no go.mod in %s (forst test packages need a local go.mod)", pkg.RelPath, modRoot)
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
			return exitErr.ExitCode(), nil
		}
		return 1, fmt.Errorf("%s: go test: %w", pkg.RelPath, err)
	}
	return 0, nil
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
