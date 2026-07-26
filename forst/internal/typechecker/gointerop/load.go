package gointerop

import (
	"forst/internal/ast"
	"forst/internal/goload"

	"golang.org/x/tools/go/packages"
)

// ImportPathsFromForstImports collects unique Go import paths from Forst import lines.
func ImportPathsFromForstImports(imports []ast.ImportNode) []string {
	pathSet := make(map[string]struct{})
	for _, imp := range imports {
		ip := goload.ImportPathFromForst(imp.Path)
		if ip != "" {
			pathSet[ip] = struct{}{}
		}
	}
	if len(pathSet) == 0 {
		return nil
	}
	paths := make([]string, 0, len(pathSet))
	for p := range pathSet {
		paths = append(paths, p)
	}
	return paths
}

// LoadPackages loads Go packages for the given import paths under moduleRoot.
func LoadPackages(moduleRoot string, paths []string, loader goload.PackagesLoader) (map[string]*packages.Package, error) {
	if moduleRoot == "" || len(paths) == 0 {
		return nil, nil
	}
	if loader == nil {
		return goload.LoadByPkgPath(moduleRoot, paths)
	}
	return goload.LoadByPkgPath(moduleRoot, paths, goload.WithPackagesLoader(loader))
}
