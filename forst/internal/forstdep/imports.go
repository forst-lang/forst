package forstdep

import (
	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/typechecker/gointerop"
)

// ImportPathsFromNodes collects Go import paths from Forst AST import lines.
func ImportPathsFromNodes(nodes []ast.Node) []string {
	var imports []ast.ImportNode
	for _, n := range nodes {
		switch v := n.(type) {
		case ast.ImportNode:
			if v.BridgeOptIn {
				continue
			}
			imports = append(imports, v)
		case ast.ImportGroupNode:
			for _, imp := range v.Imports {
				if imp.BridgeOptIn {
					continue
				}
				imports = append(imports, imp)
			}
		}
	}
	paths := gointerop.ImportPathsFromForstImports(imports)
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if goload.IsStdlibImportPath(p) {
			continue
		}
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
