package modulecheck

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/goload"
	"forst/internal/providersgraph"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// Options configures module-level Providers checking.
type Options struct {
	ModuleRoot string
	// PackageFilter limits to these Forst package names; empty means all packages under ModuleRoot.
	PackageFilter map[string]struct{}
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
	if log == nil {
		log = logrus.New()
		log.SetOutput(io.Discard)
	}
	scanRoot := filepath.Clean(opts.ModuleRoot)
	moduleRoot := goload.FindModuleRoot(scanRoot)
	modulePath := goload.ModulePath(moduleRoot)

	ftFiles, err := findForstFiles(scanRoot)
	if err != nil {
		return nil, err
	}

	parsed := make(map[string][]ast.Node)
	for _, filePath := range ftFiles {
		nodes, err := forstpkg.ParseForstFile(log, filePath)
		if err != nil {
			continue
		}
		parsed[filePath] = nodes
	}

	byPackage := make(map[string][]string)
	for path, nodes := range parsed {
		pkg := forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(nodes))
		if opts.PackageFilter != nil {
			if _, ok := opts.PackageFilter[pkg]; !ok {
				continue
			}
		}
		byPackage[pkg] = append(byPackage[pkg], path)
	}

	result := &ModuleResult{
		ModuleRoot:      moduleRoot,
		ModulePath:      modulePath,
		importPathMap:   forstpkg.BuildForstPackageImportPaths(moduleRoot, modulePath, byPackage),
		ForstPkgToFiles: byPackage,
		PerPackage:      make(map[string]*typechecker.TypeChecker),
		PerPackageNodes: make(map[string][]ast.Node),
	}

	perPkgProviders := make(map[string]map[ast.Identifier][]typechecker.ProviderSlot)

	packageNames := make([]string, 0, len(byPackage))
	for name := range byPackage {
		packageNames = append(packageNames, name)
	}
	sort.Strings(packageNames)

	for _, packageName := range packageNames {
		paths := byPackage[packageName]
		sort.Strings(paths)
		var astLists [][]ast.Node
		for _, p := range paths {
			astLists = append(astLists, parsed[p])
		}
		merged := forstpkg.MergePackageASTs(astLists)

		tc := typechecker.New(log, false)
		tc.GoWorkspaceDir = moduleRoot
		tc.SetForstPackage(packageName)
		tc.SetDeferProvidersWiringRootCheck(true)
		if importPath := result.ImportPathForForstPackage(packageName); importPath != "" {
			tc.SetSamePackageGoImportPath(importPath)
		}
		tc.SetModuleResult(result)
		if err := tc.CheckTypes(merged); err != nil {
			return nil, err
		}
		result.PerPackage[packageName] = tc
		result.PerPackageNodes[packageName] = merged
		perPkgProviders[packageName] = cloneSlots(tc.FunctionProviders)
	}

	typechecker.MergeModuleKnownRoots(result.PerPackage)
	if err := revalidateDeferredWiringKeys(result.PerPackage); err != nil {
		return nil, err
	}

	moduleGraph := providersgraph.NewModuleGraph(perPkgProviders)
	for packageName, tc := range result.PerPackage {
		for _, call := range typechecker.BuildModuleCrossCalls(packageName, tc, result.importPathMap) {
			moduleGraph.AddModuleCall(call)
		}
	}

	satisfies := providersgraph.ProviderScopeKeyPresent
	if len(result.PerPackage) > 0 {
		for _, tc := range result.PerPackage {
			satisfies = typechecker.ModuleSatisfiesFromTypeChecker(tc)
			break
		}
	}
	moduleGraph.ComputeFixedPoint(satisfies)

	for packageName, tc := range result.PerPackage {
		slots := moduleGraph.PerPackage(packageName)
		tc.SetFunctionProviders(slots)
		tc.FunctionProviders = slots
		perPkgProviders[packageName] = slots
		tc.RevalidateUnusedWiringKeysAfterModuleMerge()
	}

	for packageName, tc := range result.PerPackage {
		if err := typechecker.ValidateModuleProviders(packageName, tc, result.importPathMap, perPkgProviders); err != nil {
			return result, err
		}
	}

	return result, nil
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
