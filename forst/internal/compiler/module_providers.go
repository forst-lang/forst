package compiler

import (
	"fmt"
	"path/filepath"

	"forst/internal/ast"
	"forst/internal/forstcheck"
	"forst/internal/forstpkg"
	"forst/internal/goload"
	"forst/internal/modulecheck"
	"forst/internal/typechecker"
)

// typecheckForCompile runs module-level Providers checking when multiple Forst packages exist,
// returning the typechecker for the compiled package.
func (c *Compiler) typecheckForCompile(nodes []ast.Node) (*typechecker.TypeChecker, *modulecheck.ModuleResult, error) {
	// Explicit -root compiles use a fresh checker scoped to the package boundary (Node interop, merged examples).
	if c.Args.PackageRoot != "" {
		checker := typechecker.New(c.log, c.Args.ReportPhases)
		absRoot, err := filepath.Abs(c.Args.PackageRoot)
		if err != nil {
			return nil, nil, fmt.Errorf("package root: %w", err)
		}
		checker.NodeBoundaryRoot = absRoot
		checker.ConfigureForForstFile(c.goWorkspaceDirForCheck(), filepath.Dir(c.Args.FilePath), nodes)
		if err := checker.CheckTypes(nodes); err != nil {
			return checker, nil, err
		}
		return checker, nil, nil
	}

	moduleRoot := c.moduleRootForProvidersPass()
	modResult, err := modulecheck.CheckModuleProviders(c.log, modulecheck.Options{ModuleRoot: moduleRoot})
	if err != nil {
		return nil, modResult, err
	}
	forstPkg := forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(nodes))
	entryDir := entryDirFromArgs(c.Args)

	if modResult != nil && !c.typecheckUsesFreshEntryChecker(entryDir) {
		if tc := modResult.PerPackage[forstPkg]; tc != nil {
			// Module check used merged-package AST nodes; re-bind scopes to this compile's nodes.
			if err := forstcheck.RebindScopes(tc, nodes); err != nil {
				return tc, modResult, err
			}
			return tc, modResult, nil
		}
	}
	checker := typechecker.New(c.log, c.Args.ReportPhases)
	checker.ConfigureForForstFile(c.goWorkspaceDirForCheck(), filepath.Dir(c.Args.FilePath), nodes)
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
	return forstcheck.ModuleRootForSingleFile(c.Args.FilePath)
}

func entryDirFromArgs(args Args) string {
	if args.FilePath == "" {
		return ""
	}
	return filepath.Dir(args.FilePath)
}

// typecheckUsesFreshEntryChecker is true when modulecheck merges every same-package file
// in examples/in during the providers pass, which would desync scopes from a single-file compile.
func (c *Compiler) typecheckUsesFreshEntryChecker(entryDir string) bool {
	if c.Args.PackageRoot != "" || entryDir == "" {
		return false
	}
	modRoot := goload.FindModuleRoot(entryDir)
	if !goload.IsForstCompilerModule(modRoot) {
		return false
	}
	examplesIn, err := filepath.Abs(filepath.Join(modRoot, "..", "examples", "in"))
	if err != nil {
		return false
	}
	absEntry, err := filepath.Abs(entryDir)
	if err != nil {
		return false
	}
	return absEntry == examplesIn
}

// RebindTypecheckerScopes re-runs CheckTypes on nodes so scope stacks match the compile AST.
func RebindTypecheckerScopes(tc *typechecker.TypeChecker, nodes []ast.Node) error {
	return forstcheck.RebindScopes(tc, nodes)
}
