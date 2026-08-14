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

// forstPkgToFiles maps Forst package name -> absolute .ft file paths.
// Call ValidateGoPackageLayout before use; layout must match Go (one directory per package).
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
