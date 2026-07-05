package forstpkg

import (
	"path/filepath"
	"strings"
)

// buildImportPathsRel is overridden in tests to exercise rel-error handling.
var buildImportPathsRel = filepath.Rel

// ImportPathForDir returns the Go import path for dir under moduleRoot/modulePath.
func ImportPathForDir(moduleRoot, modulePath, dir string) (string, error) {
	moduleRoot = filepath.Clean(moduleRoot)
	rel, err := buildImportPathsRel(moduleRoot, dir)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	importPath := modulePath
	if rel != "." && rel != "" {
		importPath = modulePath + "/" + strings.TrimPrefix(rel, "/")
	}
	return importPath, nil
}

// BuildForstPackageImportPaths maps Go import paths to Forst package names for packages
// discovered under moduleRoot (modulePath is the go.mod module path, e.g. "example.com/app").
// forstPkgToFiles maps Forst package name -> absolute .ft file paths (one dir per package).
func BuildForstPackageImportPaths(moduleRoot, modulePath string, forstPkgToFiles map[string][]string) map[string]string {
	out := make(map[string]string)
	moduleRoot = filepath.Clean(moduleRoot)
	for forstPkg, files := range forstPkgToFiles {
		if len(files) == 0 {
			continue
		}
		dir := filepath.Dir(files[0])
		importPath, err := ImportPathForDir(moduleRoot, modulePath, dir)
		if err != nil {
			continue
		}
		out[importPath] = forstPkg
	}
	return out
}
