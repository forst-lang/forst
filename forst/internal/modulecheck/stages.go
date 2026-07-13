package modulecheck

import (
	"io"
	"path/filepath"
	"sort"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/goload"
	"forst/internal/providersgraph"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/packages"
)
type ModuleScan struct {
	scanRoot        string
	ModuleRoot      string
	ModulePath      string
	importPathMap   map[string]string
	ForstPkgToFiles map[string][]string
	PerPackage      map[string]*typechecker.TypeChecker
	PerPackageNodes map[string][]ast.Node
	packageNames    []string
	log             *logrus.Logger
	opts            Options
}

// ScanModule walks and parses Forst files under opts.ModuleRoot (unless ParsedFiles is set).
func ScanModule(log *logrus.Logger, opts Options) (*ModuleScan, error) {
	if log == nil {
		log = logrus.New()
		log.SetOutput(io.Discard)
	}
	scanRoot := filepath.Clean(opts.ModuleRoot)
	moduleRoot := goload.FindModuleRoot(scanRoot)
	modulePath := goload.ModulePath(moduleRoot)

	parsed := opts.ParsedFiles
	if parsed == nil {
		ftFiles, err := findForstFiles(scanRoot)
		if err != nil {
			return nil, err
		}
		parsed = forstpkg.ParseFilesLenientParallel(log, ftFiles)
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

	result := &ModuleScan{
		scanRoot:        scanRoot,
		ModuleRoot:      moduleRoot,
		ModulePath:      modulePath,
		importPathMap:   forstpkg.BuildForstPackageImportPaths(forstpkg.ForstImportPathRoot(scanRoot, moduleRoot), modulePath, byPackage),
		ForstPkgToFiles: byPackage,
		PerPackage:      make(map[string]*typechecker.TypeChecker),
		PerPackageNodes: make(map[string][]ast.Node),
		log:             log,
		opts:            opts,
	}

	packageNames := make([]string, 0, len(byPackage))
	for name := range byPackage {
		packageNames = append(packageNames, name)
	}
	sort.Strings(packageNames)
	result.packageNames = packageNames

	mergedByPkg := make(map[string][]ast.Node, len(packageNames))
	for _, packageName := range packageNames {
		paths := byPackage[packageName]
		sort.Strings(paths)
		var astLists [][]ast.Node
		for _, p := range paths {
			astLists = append(astLists, parsed[p])
		}
		mergedByPkg[packageName] = forstpkg.MergePackageASTs(astLists)
	}

	for _, packageName := range packageNames {
		tc := typechecker.New(log, false)
		tc.GoWorkspaceDir = moduleRoot
		tc.SetForstPackage(packageName)
		tc.SetDeferProvidersWiringRootCheck(true)
		if importPath := result.ImportPathForForstPackage(packageName); importPath != "" {
			tc.SetSamePackageGoImportPath(importPath)
		}
		tc.SetModuleResult(result.asModuleResult())
		result.PerPackage[packageName] = tc
		result.PerPackageNodes[packageName] = mergedByPkg[packageName]
	}

	return result, nil
}

func (s *ModuleScan) asModuleResult() *ModuleResult {
	return &ModuleResult{
		ModuleRoot:      s.ModuleRoot,
		ModulePath:      s.ModulePath,
		importPathMap:   s.importPathMap,
		ForstPkgToFiles: s.ForstPkgToFiles,
		PerPackage:      s.PerPackage,
		PerPackageNodes: s.PerPackageNodes,
	}
}

// ResolveNodeImports resolves opted-in TypeScript imports after CollectTypes.
func (s *ModuleScan) ResolveNodeImports() error {
	boundary := s.opts.BoundaryRoot
	if boundary == "" {
		boundary = s.scanRoot
	}
	for _, packageName := range s.packageNames {
		tc := s.PerPackage[packageName]
		nodes := s.PerPackageNodes[packageName]
		paths := s.ForstPkgToFiles[packageName]
		if tc == nil || len(paths) == 0 {
			continue
		}
		fileDir := filepath.Dir(paths[0])
		tc.NodeBoundaryRoot = boundary
		tc.ConfigureForForstFile(s.ModuleRoot, fileDir, nodes)
		if err := tc.ResolveNodeImportsAfterCollect(); err != nil {
			return err
		}
	}
	return nil
}

// InitAndCollectTypes runs CollectTypes for each package.
func (s *ModuleScan) InitAndCollectTypes() error {
	for _, packageName := range s.packageNames {
		if err := s.PerPackage[packageName].CollectTypes(s.PerPackageNodes[packageName]); err != nil {
			return err
		}
	}
	return nil
}

// LoadGoPackages batch-loads Go packages for module typecheckers unless SkipGoLoad is set.
func (s *ModuleScan) LoadGoPackages() error {
	if s.opts.SkipGoLoad {
		return nil
	}
	tcs := make([]*typechecker.TypeChecker, 0, len(s.packageNames))
	for _, packageName := range s.packageNames {
		tcs = append(tcs, s.PerPackage[packageName])
	}
	var loaded map[string]*packages.Package
	var err error
	if s.opts.GoLoader != nil {
		loaded, err = typechecker.BatchLoadGoPackagesForModuleWithLoader(s.ModuleRoot, tcs, s.opts.GoLoader)
	} else {
		loaded, err = typechecker.BatchLoadGoPackagesForModule(s.ModuleRoot, tcs)
	}
	if err != nil {
		s.log.WithError(err).Debug("module-wide go/packages batch load failed; Forst↔Go boundary checks may be skipped")
	}
	for _, tc := range tcs {
		tc.InitGoPackagesFromBatch(loaded)
	}
	return nil
}

// InferProviderSlots typechecks each package and returns per-package provider slots before merge.
func (s *ModuleScan) InferProviderSlots() (map[string]map[ast.Identifier][]typechecker.ProviderSlot, error) {
	perPkgProviders := make(map[string]map[ast.Identifier][]typechecker.ProviderSlot)
	for _, packageName := range s.packageNames {
		tc := s.PerPackage[packageName]
		if err := tc.InferTypes(s.PerPackageNodes[packageName]); err != nil {
			return nil, err
		}
		perPkgProviders[packageName] = cloneSlots(tc.FunctionProviders)
	}
	return perPkgProviders, nil
}

// MergeAndValidate runs provider fixed-point merge and optional validation.
func (s *ModuleScan) MergeAndValidate(perPkgProviders map[string]map[ast.Identifier][]typechecker.ProviderSlot) error {
	typechecker.MergeModuleKnownRoots(s.PerPackage)
	if err := revalidateDeferredWiringKeys(s.PerPackage); err != nil {
		return err
	}

	moduleGraph := providersgraph.NewModuleGraph(perPkgProviders)
	for packageName, tc := range s.PerPackage {
		for _, call := range typechecker.BuildModuleCrossCalls(packageName, tc, s.importPathMap) {
			moduleGraph.AddModuleCall(call)
		}
	}

	satisfies := providersgraph.ProviderScopeKeyPresent
	if len(s.PerPackage) > 0 {
		for _, tc := range s.PerPackage {
			satisfies = typechecker.ModuleSatisfiesFromTypeChecker(tc)
			break
		}
	}
	moduleGraph.ComputeFixedPoint(satisfies)

	for packageName, tc := range s.PerPackage {
		slots := moduleGraph.PerPackage(packageName)
		tc.SetFunctionProviders(slots)
		tc.FunctionProviders = slots
		perPkgProviders[packageName] = slots
		tc.RevalidateUnusedWiringKeysAfterModuleMerge()
	}

	if s.opts.SkipValidate {
		return nil
	}
	for packageName, tc := range s.PerPackage {
		if err := typechecker.ValidateModuleProviders(packageName, tc, s.importPathMap, perPkgProviders); err != nil {
			return err
		}
	}
	return nil
}

// Result returns the module result view after a full pipeline run.
func (s *ModuleScan) Result() *ModuleResult {
	return s.asModuleResult()
}

func (s *ModuleScan) ImportPathForForstPackage(forstPkg string) string {
	if forstPkg == "" {
		return ""
	}
	for importPath, pkg := range s.importPathMap {
		if pkg == forstPkg {
			return importPath
		}
	}
	return ""
}

// runModulePipeline executes scan → collect → load → infer → merge.
func runModulePipeline(log *logrus.Logger, opts Options) (*ModuleResult, error) {
	scan, err := ScanModule(log, opts)
	if err != nil {
		return nil, err
	}
	if err := scan.InitAndCollectTypes(); err != nil {
		return nil, err
	}
	if err := scan.ResolveNodeImports(); err != nil {
		return nil, err
	}
	if err := scan.LoadGoPackages(); err != nil {
		return nil, err
	}
	slots, err := scan.InferProviderSlots()
	if err != nil {
		return scan.Result(), err
	}
	if err := scan.MergeAndValidate(slots); err != nil {
		return scan.Result(), err
	}
	return scan.Result(), nil
}
