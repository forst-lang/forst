package project

import (
	"fmt"
	"path/filepath"
	"sort"

	"forst/internal/ast"
	"forst/internal/devserver"
	"forst/internal/discovery"
	"forst/internal/ftconfig"
	"forst/internal/forstpkg"
	"forst/internal/goload"
	"forst/internal/modulecheck"

	"github.com/sirupsen/logrus"
)

// DevProfile is the resolved forst dev execution profile.
type DevProfile string

const (
	DevProfileAuto     DevProfile = "auto"
	DevProfileExecutor DevProfile = "executor"
	DevProfileRuntime  DevProfile = "runtime"
)

// OpenOpts configures project.Open.
type OpenOpts struct {
	BoundaryRoot string
	ConfigPath   string
	Cwd          string
}

// Project is the unified resolution kernel for all Forst commands.
type Project struct {
	BoundaryRoot string
	ModuleRoot   string
	ModulePath   string
	Config       *ftconfig.Config
	Module       *modulecheck.ModuleResult
	log          *logrus.Logger
	profile      DevProfile
}

// Open resolves boundary, module roots, ftconfig, and modulecheck graph.
func Open(log *logrus.Logger, opts OpenOpts) (*Project, error) {
	if log == nil {
		log = logrus.New()
	}
	boundary := opts.BoundaryRoot
	if boundary == "" {
		boundary = opts.Cwd
	}
	if boundary == "" {
		boundary = "."
	}
	absBoundary, err := filepath.Abs(boundary)
	if err != nil {
		return nil, err
	}
	layout, err := goload.ResolveProjectLayout(absBoundary)
	if err != nil {
		return nil, err
	}
	cfg, err := loadConfig(opts.ConfigPath, layout.Boundary)
	if err != nil {
		return nil, err
	}
	modResult, err := modulecheck.CheckModuleProviders(log, modulecheck.Options{
		ModuleRoot:   layout.ScanRoot,
		BoundaryRoot: layout.Boundary,
	})
	if err != nil {
		return nil, fmt.Errorf("module check: %w", err)
	}
	p := &Project{
		BoundaryRoot: layout.Boundary,
		ModuleRoot:   layout.GoModRoot,
		ModulePath:   modResult.ModulePath,
		Config:       cfg,
		Module:       modResult,
		log:          log,
		profile:      resolveDevProfile(cfg),
	}
	return p, nil
}

func loadConfig(explicit, boundary string) (*ftconfig.Config, error) {
	if explicit != "" {
		return ftconfig.Load(explicit)
	}
	if found, _ := ftconfig.FindConfigFile(boundary); found != "" {
		return ftconfig.Load(found)
	}
	return ftconfig.Default(), nil
}

func resolveDevProfile(cfg *ftconfig.Config) DevProfile {
	return DevProfile(devserver.ResolveProfile(cfg))
}

// DevProfile returns the resolved dev profile.
func (p *Project) DevProfile() DevProfile {
	if p == nil {
		return DevProfileExecutor
	}
	return p.profile
}

// ForstPackages returns sorted Forst package names in scope.
func (p *Project) ForstPackages() []string {
	if p == nil || p.Module == nil {
		return nil
	}
	names := make([]string, 0, len(p.Module.ForstPkgToFiles))
	for name := range p.Module.ForstPkgToFiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// PackageUnit is a merged Forst package ready for typecheck/transform.
type PackageUnit struct {
	Name      string
	FilePaths []string
	Nodes     []ast.Node
}

// Package returns a merged package unit by Forst package name.
func (p *Project) Package(name string) (*PackageUnit, error) {
	if p == nil || p.Module == nil {
		return nil, fmt.Errorf("project not open")
	}
	paths, ok := p.Module.ForstPkgToFiles[name]
	if !ok || len(paths) == 0 {
		return nil, fmt.Errorf("package %q not found", name)
	}
	sort.Strings(paths)
	nodes := p.Module.PerPackageNodes[name]
	if nodes == nil {
		merged, _, err := forstpkg.ParseAndMergePackage(p.log, paths)
		if err != nil {
			return nil, err
		}
		nodes = merged
	}
	return &PackageUnit{
		Name:      name,
		FilePaths: paths,
		Nodes:     nodes,
	}, nil
}

// RunnableFunctions returns module-wide runnable public exports.
func (p *Project) RunnableFunctions() ([]discovery.FunctionInfo, error) {
	return discovery.CollectInvokeFunctionsFromModule(p.log, p.BoundaryRoot)
}

// FilterPackagesUnderBoundary returns packages with files under boundaryRoot.
func (p *Project) FilterPackagesUnderBoundary() map[string][]string {
	if p == nil || p.Module == nil {
		return nil
	}
	out := make(map[string][]string)
	for pkg, paths := range p.Module.ForstPkgToFiles {
		var kept []string
		for _, path := range paths {
			if underBoundary(p.BoundaryRoot, path) {
				kept = append(kept, path)
			}
		}
		if len(kept) > 0 {
			sort.Strings(kept)
			out[pkg] = kept
		}
	}
	return out
}

func underBoundary(boundary, filePath string) bool {
	rel, err := filepath.Rel(boundary, filePath)
	if err != nil {
		return false
	}
	return rel != ".." && !filepath.IsAbs(rel) && !startsWithDotDot(rel)
}

func startsWithDotDot(rel string) bool {
	return rel == ".." || len(rel) > 3 && rel[:3] == "../" || len(rel) > 4 && rel[:4] == `..\`
}
