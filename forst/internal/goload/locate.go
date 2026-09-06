package goload

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// PackageLoc is a resolved Go package directory from the module graph.
// Dir may be set even when go/packages reports load errors (Forst-only packages).
type PackageLoc struct {
	ImportPath    string
	Dir           string
	ModulePath    string
	ModuleVersion string
	ModuleDir     string
	LoadErr       string
}

// LocatePackageDirs resolves import paths to package directories under moduleRoot.
// Unlike LoadByPkgPath, this does not require a typed package (Forst-only dirs work).
// Falls back to `go list -m` when packages.Load cannot locate a Forst-only package.
func LocatePackageDirs(moduleRoot string, importPaths []string, opts ...LoadOpt) (map[string]PackageLoc, error) {
	if len(importPaths) == 0 {
		return nil, nil
	}
	cfg := resolveLoadConfig(opts)
	moduleRoot = FindModuleRoot(moduleRoot)
	if moduleRoot == "" {
		return nil, fmt.Errorf("locate packages: empty module root")
	}
	loader := cfg.loader
	if loader == nil {
		loader = packagesLoadFn
	}
	loadCfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedModule,
		Dir:  moduleRoot,
		Env:  loadPackagesEnv(moduleRoot),
	}
	pkgs, err := loader(loadCfg, importPaths...)
	if err != nil {
		return nil, err
	}
	out := make(map[string]PackageLoc, len(importPaths))
	for _, p := range pkgs {
		if p == nil {
			continue
		}
		loc := packageLocFromPackagesPkg(p)
		if loc.Dir == "" || loc.ImportPath == "" {
			continue
		}
		out[loc.ImportPath] = loc
	}
	for _, path := range importPaths {
		if _, ok := out[path]; ok {
			continue
		}
		loc, err := ResolveImportDir(moduleRoot, path)
		if err != nil {
			continue
		}
		out[path] = loc
	}
	return out, nil
}

func packageLocFromPackagesPkg(p *packages.Package) PackageLoc {
	loc := PackageLoc{
		ImportPath: pickPackagePath(p),
		Dir:        filepath.Clean(p.Dir),
	}
	if loc.Dir == "." || loc.Dir == "" {
		loc.Dir = ""
	}
	if p.Module != nil {
		loc.ModulePath = p.Module.Path
		loc.ModuleVersion = p.Module.Version
		if p.Module.Dir != "" {
			loc.ModuleDir = filepath.Clean(p.Module.Dir)
		}
	}
	if len(p.Errors) > 0 {
		msgs := make([]string, 0, len(p.Errors))
		for _, e := range p.Errors {
			if strings.TrimSpace(e.Msg) != "" {
				msgs = append(msgs, e.Msg)
			}
		}
		loc.LoadErr = strings.Join(msgs, "; ")
	}
	return loc
}

func pickPackagePath(p *packages.Package) string {
	if p == nil {
		return ""
	}
	if p.PkgPath != "" {
		return p.PkgPath
	}
	if p.Types != nil && p.Types.Path() != "" {
		return p.Types.Path()
	}
	return p.ID
}
