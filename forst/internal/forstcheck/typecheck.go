// Package forstcheck provides shared Forst file typechecking used by LSP and compiler frontends.
package forstcheck

import (
	"path/filepath"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/goload"
	"forst/internal/modulecheck"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// TypecheckFileOpts configures a single-file typecheck with optional module providers pass.
type TypecheckFileOpts struct {
	FilePath string
	Nodes    []ast.Node
}

// ModuleRootForSingleFile limits the Providers module walk for single-file analysis inside
// the compiler workspace (same rule as compile/LSP).
func ModuleRootForSingleFile(filePath string) string {
	entryDir := filepath.Dir(filePath)
	modRoot := goload.FindModuleRoot(entryDir)
	if goload.IsForstCompilerModule(modRoot) {
		return entryDir
	}
	return modRoot
}

// TypecheckFile runs module-level checking when the file lives in a multi-package Forst module,
// then typechecks nodes. Returns the entry package typechecker and optional module result.
func TypecheckFile(log *logrus.Logger, opts TypecheckFileOpts) (*typechecker.TypeChecker, *modulecheck.ModuleResult, error) {
	moduleRoot := ModuleRootForSingleFile(opts.FilePath)
	forstPkg := forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(opts.Nodes))

	modResult, err := modulecheck.CheckModuleProviders(log, modulecheck.Options{ModuleRoot: moduleRoot})
	if err == nil && modResult != nil {
		if tc := modResult.PerPackage[forstPkg]; tc != nil {
			if err := RebindScopes(tc, opts.Nodes); err != nil {
				return tc, modResult, err
			}
			return tc, modResult, nil
		}
	}

	tc := typechecker.New(log, false)
	tc.ConfigureForForstFile(moduleRoot, filepath.Dir(opts.FilePath), opts.Nodes)
	if err := tc.CheckTypes(opts.Nodes); err != nil {
		return tc, modResult, err
	}
	return tc, modResult, nil
}

// RebindScopes re-runs CheckTypes on nodes so scope stacks match the compile AST.
func RebindScopes(tc *typechecker.TypeChecker, nodes []ast.Node) error {
	savedProviders := cloneFunctionProviders(tc.FunctionProviders)
	if err := tc.CheckTypes(nodes); err != nil {
		return err
	}
	tc.SetFunctionProviders(savedProviders)
	tc.FunctionProviders = savedProviders
	return nil
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
