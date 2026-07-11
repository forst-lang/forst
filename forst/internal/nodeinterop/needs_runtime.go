package nodeinterop

import (
	"forst/internal/ast"
	"path/filepath"
	"sort"
)

// NeedsNodeRuntimeResult summarizes compile-time Node runtime requirement.
type NeedsNodeRuntimeResult struct {
	Needed  bool
	Modules []string
	Exports []ExportEntry
}

// AnalyzeNeedsNodeRuntime walks opted-in imports in nodes and collects TS module paths.
// Imports are resolved relative to boundaryRoot (the ftconfig project root).
// Exports remain empty until index data is merged via BuildManifestV1.
func AnalyzeNeedsNodeRuntime(nodes []ast.Node, boundaryRoot string) NeedsNodeRuntimeResult {
	var result NeedsNodeRuntimeResult
	moduleSet := make(map[string]struct{})

	for _, imp := range collectImportNodes(nodes) {
		if !imp.NodeOptIn {
			continue
		}
		path := normalizeImportPath(imp.Path)
		if path == "" {
			continue
		}

		if IsBareNodeSpecifier(path) {
			result.Needed = true
			moduleSet[path] = struct{}{}
			continue
		}

		moduleID, _, err := resolveTSImportUnderBoundary(boundaryRoot, path, boundaryRoot)
		if err != nil {
			continue
		}
		result.Needed = true
		moduleSet[moduleID] = struct{}{}
	}

	result.Modules = sortedKeys(moduleSet)
	return result
}

func resolveTSImportUnderBoundary(entryDir, importPath, boundaryRoot string) (moduleId string, absPath string, err error) {
	moduleId, absPath, err = ResolveTSImport(entryDir, importPath)
	if err != nil {
		return "", "", err
	}
	if boundaryRoot == "" {
		return moduleId, absPath, nil
	}
	relBoundary, relErr := moduleID(filepath.Clean(boundaryRoot), absPath)
	if relErr != nil {
		return "", "", relErr
	}
	return relBoundary, absPath, nil
}

func collectImportNodes(nodes []ast.Node) []ast.ImportNode {
	var out []ast.ImportNode
	var walk func([]ast.Node)
	walk = func(ns []ast.Node) {
		for _, node := range ns {
			switch n := node.(type) {
			case ast.ImportNode:
				out = append(out, n)
			case ast.ImportGroupNode:
				out = append(out, n.Imports...)
			case ast.FunctionNode:
				walk(n.Body)
			case *ast.FunctionNode:
				if n != nil {
					walk(n.Body)
				}
			}
		}
	}
	walk(nodes)
	return out
}

func sortedKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
