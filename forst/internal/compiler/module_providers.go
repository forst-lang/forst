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
			savedProviders := cloneFunctionProviders(tc.FunctionProviders)
			// Module check used merged-package AST nodes; re-bind scopes to this compile's nodes.
			if err := tc.CheckTypes(nodes); err != nil {
				return tc, modResult, err
			}
			tc.SetFunctionProviders(savedProviders)
			tc.FunctionProviders = savedProviders
			return tc, modResult, nil
		}
	}
	checker := typechecker.New(c.log, c.Args.ReportPhases)
	checker.GoWorkspaceDir = c.goWorkspaceDirForCheck()
	checker.SetForstPackage(forstPkg)
	if modRoot := checker.GoWorkspaceDir; modRoot != "" {
		modPath := goload.ModulePath(modRoot)
		entryDir := filepath.Dir(c.Args.FilePath)
		if importPath, err := forstpkg.ImportPathForDir(modRoot, modPath, entryDir); err == nil {
			checker.SetSamePackageGoImportPath(importPath)
		}
	}
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
	if !isCompilerWorkspaceModule(modRoot) {
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
