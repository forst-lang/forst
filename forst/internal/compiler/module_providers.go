package compiler

import (
	"os"
	"path/filepath"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/goload"
	"forst/internal/modulecheck"
	"forst/internal/typechecker"
)

// typecheckForCompile runs module-level Providers checking when multiple Forst packages exist,
// returning the typechecker for the compiled package.
func (c *Compiler) typecheckForCompile(nodes []ast.Node) (*typechecker.TypeChecker, *modulecheck.ModuleResult, error) {
	moduleRoot := c.moduleRootForProvidersPass()
	modResult, err := modulecheck.CheckModuleProviders(c.log, modulecheck.Options{ModuleRoot: moduleRoot})
	if err != nil {
		return nil, modResult, err
	}
	forstPkg := forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(nodes))
	if modResult != nil {
		if tc := modResult.PerPackage[forstPkg]; tc != nil {
			return tc, modResult, nil
		}
	}
	checker := typechecker.New(c.log, c.Args.ReportPhases)
	checker.GoWorkspaceDir = c.goWorkspaceDirForCheck()
	if err := checker.CheckTypes(nodes); err != nil {
		return checker, modResult, err
	}
	return checker, modResult, nil
}

// TypecheckForCompileEntry loads compile input AST and runs module-level typechecking.
func (c *Compiler) TypecheckForCompileEntry() (*typechecker.TypeChecker, *modulecheck.ModuleResult, error) {
	nodes, err := c.loadInputNodesForCompile()
	if err != nil {
		return nil, nil, err
	}
	return c.typecheckForCompile(nodes)
}

// moduleRootForProvidersPass limits the Providers module walk for single-file compiles
// inside the compiler repo (avoids scanning every examples/*.ft under forst/go.mod).
func (c *Compiler) moduleRootForProvidersPass() string {
	if c.Args.PackageRoot != "" {
		return goload.FindModuleRoot(c.Args.PackageRoot)
	}
	entryDir := filepath.Dir(c.Args.FilePath)
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
