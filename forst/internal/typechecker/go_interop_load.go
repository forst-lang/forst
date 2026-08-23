package typechecker

import (
	"fmt"

	"forst/internal/goload"

	"golang.org/x/tools/go/packages"
)

// recordGoPackagesLoadFailure stores a go/packages load error for each import path in paths.
func (tc *TypeChecker) recordGoPackagesLoadFailure(paths []string, err error) {
	if tc == nil || err == nil || len(paths) == 0 {
		return
	}
	if tc.goImportLoadErrors == nil {
		tc.goImportLoadErrors = make(map[string]error)
	}
	for _, p := range paths {
		if p != "" {
			tc.goImportLoadErrors[p] = err
		}
	}
}

func (tc *TypeChecker) goImportLoadErrorForPath(path string) error {
	if tc == nil || tc.goImportLoadErrors == nil || path == "" {
		return nil
	}
	return tc.goImportLoadErrors[path]
}

func (tc *TypeChecker) goImportLoadErrorForLocal(local string) error {
	if tc == nil || local == "" {
		return nil
	}
	path, ok := tc.ImportPathForLocal(local)
	if !ok {
		return nil
	}
	return tc.goImportLoadErrorForPath(path)
}

// RecordUnloadedGoImportPaths stores load errors for import paths missing from a batch result.
func (tc *TypeChecker) RecordUnloadedGoImportPaths(loaded map[string]*packages.Package, batchErr error) {
	tc.recordUnloadedGoImportPaths(loaded, batchErr)
}

// recordUnloadedGoImportPaths stores a load error for each referenced import path
// that is missing from loaded or failed PackageLoadOK after a batch load.
func (tc *TypeChecker) recordUnloadedGoImportPaths(loaded map[string]*packages.Package, batchErr error) {
	if tc == nil {
		return
	}
	var missing []string
	for _, imp := range tc.imports {
		if imp.Alias != nil && string(imp.Alias.ID) == "." {
			continue
		}
		path := goload.ImportPathFromForst(imp.Path)
		if path == "" {
			continue
		}
		pkg, ok := loaded[path]
		if !ok || !goload.PackageLoadOK(pkg, path) {
			missing = append(missing, path)
		}
	}
	if len(missing) == 0 {
		return
	}
	err := batchErr
	if err == nil {
		err = fmt.Errorf("Go package types not loaded")
	}
	tc.recordGoPackagesLoadFailure(missing, err)
}
