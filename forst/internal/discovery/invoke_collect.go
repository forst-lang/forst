package discovery

import (
	"io"

	"forst/internal/ast"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// CollectInvokeFunctionsFromNodes returns runnable public functions from parsed AST nodes.
func CollectInvokeFunctionsFromNodes(nodes []ast.Node, tc *typechecker.TypeChecker) []FunctionInfo {
	log := logrus.New()
	log.SetOutput(io.Discard)
	d := NewDiscoverer("", log, nil)
	pkg := extractPackageNameFromNodes(nodes)
	if pkg == "" {
		pkg = "main"
	}
	fns := make(map[string]FunctionInfo)
	for _, node := range nodes {
		d.extractFunctionsFromNode(node, pkg, "", fns, tc)
	}
	var out []FunctionInfo
	for _, fn := range fns {
		if fn.Runnable && fn.Name != "main" {
			out = append(out, fn)
		}
	}
	return out
}

func extractPackageNameFromNodes(nodes []ast.Node) string {
	for _, node := range nodes {
		if pkgNode, ok := node.(ast.PackageNode); ok {
			return string(pkgNode.Ident.ID)
		}
	}
	return ""
}
