package typechecker

import "forst/internal/ast"

// partitionTopLevelForCollect returns a reordering of top-level nodes for the collect pass only.
// Types and type guards are collected before functions so user-defined types used in signatures
// are registered in tc.Defs before registerFunction runs, regardless of source file order or
// merge order (multiple .ft files concatenated arbitrarily).
//
// The inference pass still uses the original slice order (see CheckTypes).
func partitionTopLevelForCollect(nodes []ast.Node) []ast.Node {
	var pkg, imports, typeDefs, packageVars, typeGuards, funcs, rest []ast.Node
	for _, node := range nodes {
		switch n := node.(type) {
		case ast.PackageNode:
			pkg = append(pkg, node)
		case ast.ImportNode:
			imports = append(imports, node)
		case ast.ImportGroupNode:
			imports = append(imports, node)
		case ast.TypeDefNode:
			typeDefs = append(typeDefs, node)
		case ast.AssignmentNode:
			if n.IsPackageLevel {
				packageVars = append(packageVars, node)
			} else {
				rest = append(rest, node)
			}
		case ast.TypeGuardNode, *ast.TypeGuardNode:
			typeGuards = append(typeGuards, node)
		case ast.FunctionNode, *ast.FunctionNode:
			funcs = append(funcs, node)
		default:
			rest = append(rest, node)
		}
	}
	out := make([]ast.Node, 0, len(nodes))
	out = append(out, pkg...)
	out = append(out, imports...)
	out = append(out, typeDefs...)
	out = append(out, packageVars...)
	out = append(out, typeGuards...)
	out = append(out, funcs...)
	out = append(out, rest...)
	return out
}
