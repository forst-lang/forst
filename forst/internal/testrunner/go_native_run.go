package testrunner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"forst/internal/goload"
	"forst/internal/gowork"
	"forst/internal/project"
)

// GoTestPackage is a Go-only package (no *_test.ft) that contains native Go tests (*_test.go).
type GoTestPackage struct {
	Dir        string // absolute directory path
	ImportPath string // full import path (e.g. module/pkg)
	RelPath    string // slash-separated relative path from module root
}

// DiscoverGoTestPackages finds package directories under moduleRoot that have *_test.go but no *_test.ft.
func DiscoverGoTestPackages(moduleRoot, modulePath string, skipDirs map[string]struct{}) ([]GoTestPackage, error) {
	if moduleRoot == "" || modulePath == "" {
		return nil, nil
	}
	var out []GoTestPackage
	err := filepath.WalkDir(moduleRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		base := d.Name()
		if base == "vendor" || base == "node_modules" || base == ".git" || base == ".forst" {
			return filepath.SkipDir
		}
		if _, skip := skipDirs[path]; skip {
			return nil
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}

		hasGoTest, hasFtTest := false, false
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasSuffix(name, "_test.go") {
				hasGoTest = true
			}
			if strings.HasSuffix(name, "_test.ft") {
				hasFtTest = true
			}
		}

		if hasGoTest && !hasFtTest {
			rel, err := filepath.Rel(moduleRoot, path)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			imp := modulePath
			if rel != "." {
				imp = modulePath + "/" + rel
			}
			out = append(out, GoTestPackage{
				Dir:        path,
				ImportPath: imp,
				RelPath:    rel,
			})
		}
		return nil
	})
	return out, err
}

// RunGoTestOnNativePackage executes go test directly on a Go-only package directory in the user module context.
func RunGoTestOnNativePackage(proj *project.Project, goPkg GoTestPackage, replaces []gowork.PackageReplace, goTestArgs []string) (ExitCode, error) {
	compilerMod := goload.ForstCompilerModuleRoot()
	tempRunDir, err := os.MkdirTemp("", "forst-gotest-*")
	if err != nil {
		return ExitError, err
	}
	defer func() { _ = os.RemoveAll(tempRunDir) }()

	goModPath := filepath.Join(tempRunDir, "go.mod")
	if err := gowork.WriteTestGoMod(goModPath, compilerMod, replaces, goPkg.ImportPath); err != nil {
		return ExitError, err
	}

	plan := gowork.LinkPlan{Mode: gowork.LinkReplace}
	args := []string{"test"}
	args = append(args, goTestArgs...)
	args = append(args, goPkg.ImportPath)

	cmd := exec.Command("go", args...)
	cmd.Dir = tempRunDir
	cmd.Env = gowork.ChildEnv(os.Environ(), plan, proj.BoundaryRoot)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := currentExecGoTest()(cmd); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() >= 0 {
			return ExitCode(exitErr.ExitCode()), nil
		}
		return ExitError, fmt.Errorf("%s: go test: %w", goPkg.RelPath, err)
	}
	return ExitSuccess, nil
}
