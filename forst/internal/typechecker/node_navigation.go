package typechecker

import (
	"strings"

	"forst/internal/bridgeinterop"
)

// NodeImportAbsPath returns the absolute path to the resolved TS/JS module for a JS import local name.
func (tc *TypeChecker) NodeImportAbsPath(local string) (string, bool) {
	mod, ok := tc.nodeModuleForLocal(local)
	if !ok || mod.AbsPath == "" {
		return "", false
	}
	return mod.AbsPath, true
}

// NodeImportPathAbsPath returns the absolute path for a JS import path string literal value.
func (tc *TypeChecker) NodeImportPathAbsPath(importPath string) (string, bool) {
	if tc == nil || strings.TrimSpace(importPath) == "" {
		return "", false
	}
	for _, binding := range tc.nodeImportsByLocal {
		if binding.Import.Path != importPath || binding.AbsPath == "" {
			continue
		}
		return binding.AbsPath, true
	}
	return "", false
}

// NodeExportDefinitionLocation returns the absolute path and source span for a node export.
// When the index has no definition span, ok is true with a zero loc (open module file).
func (tc *TypeChecker) NodeExportDefinitionLocation(moduleLocal, exportName string) (absPath string, loc bridgeinterop.IndexSourceLocation, ok bool) {
	if tc == nil || moduleLocal == "" || exportName == "" {
		return "", bridgeinterop.IndexSourceLocation{}, false
	}
	mod, ok := tc.nodeModuleForLocal(moduleLocal)
	if !ok || mod.Index == nil {
		return "", bridgeinterop.IndexSourceLocation{}, false
	}
	exp, ok := mod.Index.ExportByName(exportName)
	if !ok {
		return "", bridgeinterop.IndexSourceLocation{}, false
	}
	abs, ok := bridgeinterop.DefinitionAbsPath(tc.nodeBoundaryRoot(), mod.ModuleID, mod.AbsPath, exp.Definition)
	if !ok {
		return "", bridgeinterop.IndexSourceLocation{}, false
	}
	if exp.Definition != nil {
		loc = *exp.Definition
	}
	return abs, loc, true
}
