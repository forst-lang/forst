package testrunner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"forst/internal/codegen/layout"
	"forst/internal/compileplan"
	"forst/internal/forstpkg"
	"forst/internal/goload"
	"forst/internal/gowork"
	"forst/internal/modulecheck"
	"forst/internal/project"
)

// RunWithProject executes tests via unified project resolution and session-scoped .forst/gen layout.
func RunWithProject(proj *project.Project, opts Options) (ExitCode, error) {
	if proj == nil {
		return ExitError, fmt.Errorf("project is nil")
	}
	pkgs, err := DiscoverPackages(proj.ModuleRoot, opts.Paths)
	if err != nil {
		return ExitError, err
	}
	runID, err := os.MkdirTemp("", "forst-test-*")
	if err != nil {
		return ExitError, err
	}
	defer func() { _ = os.RemoveAll(runID) }()

	layoutRoot := layout.NewRoot(proj.BoundaryRoot)
	modResult := proj.Module

	var failed bool
	testDirs := make(map[string]struct{}, len(pkgs))
	for _, p := range pkgs {
		testDirs[p.Dir] = struct{}{}
	}

	libReplaces := make(map[string]string)
	if modResult != nil {
		var emitErr error
		libReplaces, emitErr = emitLibShimsSession(layoutRoot, runID, modResult, testDirs, opts, proj)
		if emitErr != nil {
			return ExitFailure, emitErr
		}
	}

	for _, pkg := range pkgs {
		code, err := runPackageTestsSession(proj, layoutRoot, runID, pkg, modResult, libReplaces, opts)
		if err != nil {
			return code, err
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

func emitLibShimsSession(layoutRoot layout.Root, runID string, modResult *modulecheck.ModuleResult, skipDirs map[string]struct{}, opts Options, proj *project.Project) (map[string]string, error) {
	emit := EmitOptions{ExportStructFields: opts.ExportStructFields}
	replaces := make(map[string]string)
	for forstPkg, paths := range modResult.ForstPkgToFiles {
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
		unit, err := proj.Package(forstPkg)
		if err != nil {
			continue
		}
		suffix := importPathSuffix(proj, forstPkg)
		shimDir := filepath.Dir(layoutRoot.LibShim(runID, suffix))
		shimPath := filepath.Join(shimDir, layout.FileLibShim)
		if err := os.MkdirAll(shimDir, 0o755); err != nil {
			return nil, err
		}
		code, err := compileplan.EmitGo(opts.Log, compileplan.Plan{
			Project:            proj,
			ForstPackage:       forstPkg,
			Nodes:              unit.Nodes,
			Checker:            modResult.PerPackage[forstPkg],
			Module:             modResult,
			ExportStructFields: emit.ExportStructFields,
		}, compileplan.EmitLibShim)
		if err != nil {
			return nil, fmt.Errorf("lib shim %s: %w", forstPkg, err)
		}
		if err := os.WriteFile(shimPath, []byte(code), 0o644); err != nil {
			return nil, err
		}
		imp := modResult.ImportPathForForstPackage(forstPkg)
		if imp == "" {
			imp = proj.ModulePath + "/" + suffix
		}
		if err := writePackageGoMod(shimDir, imp); err != nil {
			return nil, err
		}
		replaces[imp] = shimDir
	}
	return replaces, nil
}

func runPackageTestsSession(proj *project.Project, layoutRoot layout.Root, runID string, pkg PackageUnderTest, modResult *modulecheck.ModuleResult, libReplaces map[string]string, opts Options) (ExitCode, error) {
	emit := EmitOptions{ExportStructFields: opts.ExportStructFields}

	forstPkg := "main"
	if modResult != nil {
		merged, _, err := forstpkg.ParseAndMergePackage(opts.Log, pkg.FtPaths)
		if err == nil {
			forstPkg = forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(merged))
		}
	}

	var code string
	var err error
	if modResult != nil && modResult.PerPackage[forstPkg] != nil {
		unit, pkgErr := proj.Package(forstPkg)
		if pkgErr != nil {
			return ExitFailure, fmt.Errorf("%s: %w", pkg.RelPath, pkgErr)
		}
		code, err = compileplan.EmitGo(opts.Log, compileplan.Plan{
			Project:            proj,
			ForstPackage:       forstPkg,
			Nodes:              unit.Nodes,
			Checker:            modResult.PerPackage[forstPkg],
			Module:             modResult,
			ExportStructFields: emit.ExportStructFields,
		}, compileplan.EmitLibShim)
	} else {
		code, err = emitPackageGo(proj.ModuleRoot, pkg, modResult, emit, opts.Log)
	}
	if err != nil {
		return ExitFailure, fmt.Errorf("%s: %w", pkg.RelPath, err)
	}

	testPaths := layoutRoot.TestRun(runID, pkg.RelPath)
	testDir := filepath.Dir(testPaths.TestFile)
	testImport := testImportPath(proj, forstPkg, pkg.RelPath)
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		return ExitFailure, err
	}
	if err := os.WriteFile(testPaths.TestFile, []byte(code), 0o644); err != nil {
		return ExitFailure, err
	}
	if err := writePackageGoMod(testDir, testImport); err != nil {
		return ExitFailure, err
	}

	replaces := buildTestReplaces(testImport, testDir, libReplaces)
	return runGoTestSandbox(proj, testPaths, testImport, replaces, opts.GoTestArgs, pkg.RelPath)
}

func buildTestReplaces(testImport, testDir string, libReplaces map[string]string) []gowork.PackageReplace {
	var out []gowork.PackageReplace
	out = append(out, gowork.PackageReplace{ImportPath: testImport, Dir: testDir})
	for imp, dir := range libReplaces {
		if imp == testImport {
			continue
		}
		out = append(out, gowork.PackageReplace{ImportPath: imp, Dir: dir})
	}
	return out
}

func testImportPath(proj *project.Project, forstPkg, relPkg string) string {
	if proj != nil && proj.Module != nil {
		if imp := proj.Module.ImportPathForForstPackage(forstPkg); imp != "" {
			return imp
		}
	}
	if proj != nil && proj.ModulePath != "" {
		if relPkg != "" {
			return proj.ModulePath + "/" + relPkg
		}
		return proj.ModulePath
	}
	return forstPkg
}

func runGoTestSandbox(proj *project.Project, paths layout.TestPaths, testImport string, replaces []gowork.PackageReplace, goTestArgs []string, relPath string) (ExitCode, error) {
	if err := os.MkdirAll(paths.ModDir, 0o755); err != nil {
		return ExitFailure, err
	}
	compilerMod := goload.ForstCompilerModuleRoot()
	if err := gowork.WriteTestGoMod(paths.GoMod, compilerMod, replaces, testImport); err != nil {
		return ExitFailure, err
	}
	plan, _ := gowork.PlanForRun(proj.BoundaryRoot, paths.ModDir, compilerMod != "")
	args := []string{"test"}
	args = append(args, goTestArgs...)
	args = append(args, testImport)

	cmd := exec.Command("go", args...)
	cmd.Dir = paths.ModDir
	cmd.Env = gowork.ChildEnv(os.Environ(), plan, proj.BoundaryRoot)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := currentExecGoTest()(cmd); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() >= 0 {
			return ExitCode(exitErr.ExitCode()), nil
		}
		return ExitError, fmt.Errorf("%s: go test: %w", relPath, err)
	}
	return ExitSuccess, nil
}

func importPathSuffix(proj *project.Project, forstPkg string) string {
	if proj == nil || proj.Module == nil {
		return forstPkg
	}
	imp := proj.Module.ImportPathForForstPackage(forstPkg)
	if imp == "" {
		return forstPkg
	}
	prefix := proj.ModulePath + "/"
	return strings.TrimPrefix(imp, prefix)
}

func writePackageGoMod(dir, modulePath string) error {
	if modulePath == "" {
		return nil
	}
	path := filepath.Join(dir, "go.mod")
	content := "module " + modulePath + "\n\ngo 1.26.0\n"
	return os.WriteFile(path, []byte(content), 0o644)
}
