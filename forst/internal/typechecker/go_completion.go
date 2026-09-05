package typechecker

import (
	"go/types"
	"sort"
	"strings"

	"forst/internal/typechecker/gointerop"
)

// ListExportedNamesForImportLocal returns a sorted list of exported identifiers for an imported Go package local name.
// Matches prefix case-insensitively when prefix is provided.
func (tc *TypeChecker) ListExportedNamesForImportLocal(local, prefix string) []string {
	if tc == nil || local == "" {
		return nil
	}
	gp := tc.goPackageForImportLocal(local)
	if gp == nil {
		return nil
	}
	scope := gp.Scope()
	if scope == nil {
		return nil
	}

	pl := strings.ToLower(prefix)
	var out []string
	for _, name := range scope.Names() {
		if !gointerop.IdentifierExported(name) {
			continue
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), pl) {
			continue
		}
		obj := scope.Lookup(name)
		if obj == nil {
			continue
		}
		// Skip unexported interface/struct detail names if any
		if !obj.Exported() {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// GoExportObjectForImportLocal looks up a top-level exported Go types.Object in an imported Go package.
func (tc *TypeChecker) GoExportObjectForImportLocal(local, name string) types.Object {
	if tc == nil || local == "" || name == "" {
		return nil
	}
	gp := tc.goPackageForImportLocal(local)
	if gp == nil {
		return nil
	}
	scope := gp.Scope()
	if scope == nil {
		return nil
	}
	obj := scope.Lookup(name)
	if obj != nil && obj.Exported() {
		return obj
	}
	return nil
}
