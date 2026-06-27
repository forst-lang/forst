package compiler

import (
	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/modulecheck"
	"forst/internal/typechecker"
)

// typecheckForCompile runs module-level Providers checking when multiple Forst packages exist,
// returning the typechecker for the compiled package.
func (c *Compiler) typecheckForCompile(nodes []ast.Node) (*typechecker.TypeChecker, *modulecheck.ModuleResult, error) {
	moduleRoot := c.goWorkspaceDirForCheck()
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
	checker.GoWorkspaceDir = moduleRoot
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
