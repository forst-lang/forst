package typechecker

import "forst/internal/ast"

// partitionTopLevelForCollect returns a reordering of top-level nodes for the collect pass only.
// Types and type guards are collected before functions so user-defined types used in signatures
// are registered in tc.Defs before registerFunction runs, regardless of source file order or
// merge order (multiple .ft files concatenated arbitrarily).
//
// The inference pass still uses the original slice order (see CheckTypes).
func partitionTopLevelForCollect(nodes []ast.Node) []ast.Node {
	var pkg, imports, typeDefs, typeGuards, funcs, rest []ast.Node
	for _, n := range nodes {
		switch n := n.(type) {
		case ast.PackageNode:
			pkg = append(pkg, n)
		case ast.ImportNode:
			imports = append(imports, n)
		case ast.ImportGroupNode:
			imports = append(imports, n)
		case ast.TypeDefNode:
			typeDefs = append(typeDefs, n)
		case ast.TypeGuardNode:
			typeGuards = append(typeGuards, n)
		case ast.FunctionNode:
			funcs = append(funcs, n)
		case *ast.FunctionNode:
			funcs = append(funcs, n)
		default:
			rest = append(rest, n)
		}
	}
	out := make([]ast.Node, 0, len(nodes))
	out = append(out, pkg...)
	out = append(out, imports...)
	out = append(out, typeDefs...)
	out = append(out, typeGuards...)
	out = append(out, funcs...)
	out = append(out, rest...)
	return out
}
