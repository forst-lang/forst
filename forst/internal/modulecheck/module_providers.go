package modulecheck

import (
	"os"
	"path/filepath"
	"strings"

	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// Options configures module-level Providers checking.
type Options struct {
	ModuleRoot string
	// PackageFilter limits to these Forst package names; empty means all packages under ModuleRoot.
	PackageFilter map[string]struct{}
	// SkipGoLoad skips go/packages batch load (Forst-only provider tests).
	SkipGoLoad bool
	// SkipValidate skips final ValidateModuleProviders (infer/merge-only tests).
	SkipValidate bool
	// ParsedFiles bypasses disk walk when non-nil (path -> AST nodes).
	ParsedFiles map[string][]ast.Node
	// BoundaryRoot is the ftconfig project root for node import resolution (defaults to ModuleRoot).
	BoundaryRoot string
	// GoLoader overrides go/packages load for tests; nil uses default.
	GoLoader goload.PackagesLoader
}

// ModuleResult holds per-package typecheckers after module-level Providers merge.
type ModuleResult struct {
	ModuleRoot      string
	ModulePath      string
	importPathMap   map[string]string
	ForstPkgToFiles map[string][]string
	PerPackage      map[string]*typechecker.TypeChecker
	PerPackageNodes map[string][]ast.Node
}

// CheckModuleProviders typechecks all Forst packages in a module and runs cross-package fixed-point.
func CheckModuleProviders(log *logrus.Logger, opts Options) (*ModuleResult, error) {
	return runModulePipeline(log, opts)
}

// ImportPathToForstPkg returns the import path → Forst package map.
func (r *ModuleResult) ImportPathToForstPkg() map[string]string {
	if r == nil {
		return nil
	}
	return r.importPathMap
}

// ImportPathForForstPackage returns the Go import path for a Forst package name, or "" if unknown.
func (r *ModuleResult) ImportPathForForstPackage(forstPkg string) string {
	if r == nil || forstPkg == "" {
		return ""
	}
	for importPath, pkg := range r.importPathMap {
		if pkg == forstPkg {
			return importPath
		}
	}
	return ""
}

// ForstPackageTypeChecker returns the typechecker for a Forst package name.
func (r *ModuleResult) ForstPackageTypeChecker(pkg string) *typechecker.TypeChecker {
	if r == nil {
		return nil
	}
	return r.PerPackage[pkg]
}

func findForstFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			if path != root {
				if _, statErr := os.Stat(filepath.Join(path, "go.mod")); statErr == nil {
					return filepath.SkipDir
				}
				// Nested ftconfig.json marks an isolated project boundary (node-interop, tictactoe, …).
				if _, statErr := os.Stat(filepath.Join(path, "ftconfig.json")); statErr == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if filepath.Ext(path) == ".ft" {
			if strings.HasSuffix(path, ".skip.ft") {
				return nil
			}
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

func cloneSlots(src map[ast.Identifier][]typechecker.ProviderSlot) map[ast.Identifier][]typechecker.ProviderSlot {
	if len(src) == 0 {
		return make(map[ast.Identifier][]typechecker.ProviderSlot)
	}
	out := make(map[ast.Identifier][]typechecker.ProviderSlot, len(src))
	for fn, slots := range src {
		copied := make([]typechecker.ProviderSlot, len(slots))
		copy(copied, slots)
		out[fn] = copied
	}
	return out
}
