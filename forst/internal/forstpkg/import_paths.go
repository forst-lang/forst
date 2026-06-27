package forstpkg

import (
	"path/filepath"
	"strings"
)

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
		rel, err := filepath.Rel(moduleRoot, dir)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		importPath := modulePath
		if rel != "." && rel != "" {
			importPath = modulePath + "/" + strings.TrimPrefix(rel, "/")
		}
		out[importPath] = forstPkg
	}
	return out
}
