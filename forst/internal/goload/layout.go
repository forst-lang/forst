package goload

import (
	"path/filepath"
	"strings"

	"forst/internal/forstpkg"
)

// ProjectLayout describes how a Forst project boundary maps to Go module layout.
type ProjectLayout struct {
	Boundary       string // ftconfig / -root project root
	ScanRoot       string // where to walk for .ft files
	GoModRoot      string // directory containing go.mod (may be .forst-gomod)
	ReplaceBase    string // base for replace => paths in user go.mod
	ImportPathRoot string // base for Forst Go import paths
}

// ResolveProjectLayout resolves boundary, scan root, and go.mod root for a project boundary.
func ResolveProjectLayout(boundary string) (ProjectLayout, error) {
	abs, err := filepath.Abs(boundary)
	if err != nil {
		return ProjectLayout{}, err
	}
	goModRoot := FindModuleRoot(abs)
	if goModRoot == "" {
		goModRoot = abs
	}
	scanRoot := ScanRootForPackageRoot(abs)
	return ProjectLayout{
		Boundary:       abs,
		ScanRoot:       scanRoot,
		GoModRoot:      goModRoot,
		ReplaceBase:    GoModReplaceBase(goModRoot),
		ImportPathRoot: forstpkg.ForstImportPathRoot(abs, goModRoot),
	}, nil
}

// ScanRootForPackageRoot returns where modulecheck should walk for .ft files when compiling
// with -root packageRoot. Nested entries inside a Go module scan from the module root;
// .forst-gomod layouts scan from the project boundary.
func ScanRootForPackageRoot(packageRoot string) string {
	packageRoot = filepath.Clean(packageRoot)
	goModRoot := FindModuleRoot(packageRoot)
	if goModRoot == "" || goModRoot == packageRoot {
		return packageRoot
	}
	rel, err := filepath.Rel(goModRoot, packageRoot)
	if err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
		return goModRoot
	}
	return packageRoot
}

// GoModReplaceBase is the directory relative to which replace => paths are resolved.
// .forst-gomod/go.mod paths are authored from the project boundary root, not from .forst-gomod/.
func GoModReplaceBase(moduleRoot string) string {
	moduleRoot = filepath.Clean(moduleRoot)
	if filepath.Base(moduleRoot) == forstGoModDir {
		return filepath.Dir(moduleRoot)
	}
	return moduleRoot
}
