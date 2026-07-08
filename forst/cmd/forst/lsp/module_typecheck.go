package lsp

import (
	"os"
	"path/filepath"

	"forst/internal/ast"
	"forst/internal/compiler"
	"forst/internal/forstpkg"
	"forst/internal/goload"
	"forst/internal/modulecheck"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// moduleRootForLSPTypecheck limits the Providers module walk for single-file LSP analysis
// inside the compiler workspace (same rule as compile) so editing one example does not scan
// every .ft under forst/go.mod.
func moduleRootForLSPTypecheck(filePath string) string {
	entryDir := filepath.Dir(filePath)
	modRoot := goload.FindModuleRoot(entryDir)
	if isCompilerWorkspaceModule(modRoot) {
		return entryDir
	}
	return modRoot
}

func isCompilerWorkspaceModule(modRoot string) bool {
	if modRoot == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(modRoot, "cmd", "forst"))
	return err == nil
}

// typecheckForLSP runs module-level checking when the file lives in a multi-package Forst
// module so cross-package imports resolve via Forst siblings (not emitted Go stubs).
func typecheckForLSP(log *logrus.Logger, filePath string, nodes []ast.Node) (*typechecker.TypeChecker, error) {
	moduleRoot := moduleRootForLSPTypecheck(filePath)
	forstPkg := forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(nodes))

	modResult, err := modulecheck.CheckModuleProviders(log, modulecheck.Options{ModuleRoot: moduleRoot})
	if err == nil && modResult != nil {
		if tc := modResult.PerPackage[forstPkg]; tc != nil {
			if err := compiler.RebindTypecheckerScopes(tc, nodes); err != nil {
				return tc, err
			}
			return tc, nil
		}
	}

	tc := typechecker.New(log, false)
	tc.ConfigureForForstFile(moduleRoot, filepath.Dir(filePath), nodes)
	if err := tc.CheckTypes(nodes); err != nil {
		return tc, err
	}
	return tc, nil
}
