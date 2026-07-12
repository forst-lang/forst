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

// ForstImportPathRoot returns the filesystem root used to build Go import paths for Forst packages.
// When go.mod lives in boundary/.forst-gomod, Forst sources are under boundary, not under .forst-gomod.
func ForstImportPathRoot(boundaryRoot, moduleRoot string) string {
	boundaryRoot = filepath.Clean(boundaryRoot)
	moduleRoot = filepath.Clean(moduleRoot)
	if boundaryRoot == "" {
		return moduleRoot
	}
	if moduleRoot == "" {
		return boundaryRoot
	}
	if moduleRoot == filepath.Join(boundaryRoot, ".forst-gomod") {
		return boundaryRoot
	}
	return moduleRoot
}

// BuildForstPackageImportPaths maps Go import paths to Forst package names for packages
// discovered under moduleRoot (modulePath is the go.mod module path, e.g. "example.com/app").
// forstPkgToFiles maps Forst package name -> absolute .ft file paths (one dir per package).
//
// When a package lives in parent/{pkg}.ft (directory name differs from package name), both
// the directory import path and parent/{pkg} are registered so imports like
// "example.com/app/internal/version" resolve to package version in internal/version.ft.
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
		if alt := packageNamedFileImportPath(importPath, dir, forstPkg, files); alt != "" {
			out[alt] = forstPkg
		}
	}
	return out
}

// packageNamedFileImportPath returns modulePath/.../dir/pkg when files include pkg.ft but dir != pkg.
func packageNamedFileImportPath(dirImportPath, dir, forstPkg string, files []string) string {
	if dirImportPath == "" || forstPkg == "" {
		return ""
	}
	if filepath.Base(dir) == forstPkg {
		return ""
	}
	namedFile := forstPkg + ".ft"
	for _, file := range files {
		if filepath.Base(file) == namedFile {
			return dirImportPath + "/" + forstPkg
		}
	}
	return ""
}
