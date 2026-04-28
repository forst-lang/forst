package discovery

import (
	"strings"
	"unicode"

	"forst/internal/ast"
	"forst/internal/typechecker"
)

// discoverFunctionsInParsedFile collects public functions from an already-parsed file using a
// typechecker that was run on the merged package AST (cross-file types and imports).
func (d *Discoverer) discoverFunctionsInParsedFile(nodes []ast.Node, filePath, packageName string, tc *typechecker.TypeChecker) map[string]FunctionInfo {
	fileFunctions := make(map[string]FunctionInfo)
	d.extractFunctionsFromNodes(nodes, packageName, filePath, fileFunctions, tc)
	return fileFunctions
}

// extractPackageNameFromAST extracts the package name from AST nodes
func (d *Discoverer) extractPackageNameFromAST(nodes []ast.Node) string {
	for _, node := range nodes {
		if pkgNode, ok := node.(ast.PackageNode); ok {
			return string(pkgNode.Ident.ID)
		}
	}
	return ""
}

// extractFunctionsFromNodes extracts public functions from AST nodes
func (d *Discoverer) extractFunctionsFromNodes(nodes []ast.Node, packageName, filePath string, functions map[string]FunctionInfo, tc *typechecker.TypeChecker) {
	d.log.Tracef("Processing %d AST nodes for package %s in file %s", len(nodes), packageName, filePath)
	for i, node := range nodes {
		d.log.Tracef("Processing node %d: %T", i, node)
		d.extractFunctionsFromNode(node, packageName, filePath, functions, tc)
	}
}

// extractFunctionsFromNode extracts public functions from a single AST node
func (d *Discoverer) extractFunctionsFromNode(node ast.Node, packageName, filePath string, functions map[string]FunctionInfo, tc *typechecker.TypeChecker) {
	switch n := node.(type) {
	case ast.FunctionNode:
		d.extractFunctionsFromNode(&n, packageName, filePath, functions, tc)
	case *ast.FunctionNode:
		d.log.Tracef("Found function node: %s", n.Ident.ID)
		if len(n.Ident.ID) > 0 && unicode.IsUpper(rune(n.Ident.ID[0])) {
			d.log.Tracef("Function %s is public (starts with uppercase)", n.Ident.ID)
			fnInfo := FunctionInfo{
				Package:           packageName,
				Name:              string(n.Ident.ID),
				SupportsStreaming: d.analyzeStreamingSupport(n, tc),
				FilePath:          filePath,
			}
			for _, param := range n.Params {
				fnInfo.Parameters = append(fnInfo.Parameters, ParameterInfo{
					Name: param.GetIdent(),
					Type: d.resolveTypeName(param.GetType(), tc),
				})
			}

			returnTypes := d.resolveFunctionReturnTypes(n, tc)
			d.applyFunctionReturnMetadata(&fnInfo, returnTypes, tc)
			d.applyGatewayMetadata(&fnInfo, n, tc)
			fnInfo.InputType = d.determineInputType(fnInfo.Parameters)
			fnInfo.OutputType = fnInfo.ReturnType
			functions[string(n.Ident.ID)] = fnInfo
			d.log.Tracef("Discovered public function: %s.%s", packageName, n.Ident.ID)
		} else {
			d.log.Tracef("Function %s is private (starts with lowercase)", n.Ident.ID)
		}
	default:
		d.log.Tracef("Node type %T is not a function", node)
	}
}

func (d *Discoverer) resolveFunctionReturnTypes(fn *ast.FunctionNode, tc *typechecker.TypeChecker) []ast.TypeNode {
	if tc != nil {
		if sig, exists := tc.Functions[fn.Ident.ID]; exists && len(sig.ReturnTypes) > 0 {
			d.log.Debugf("Using typechecker's inferred return types for function %s: %v", fn.Ident.ID, sig.ReturnTypes)
			return sig.ReturnTypes
		}
		d.log.Debugf("Using parser's return types for function %s: %v", fn.Ident.ID, fn.ReturnTypes)
		return fn.ReturnTypes
	}
	d.log.Debugf("No typechecker available, using parser's return types for function %s: %v", fn.Ident.ID, fn.ReturnTypes)
	return fn.ReturnTypes
}

func (d *Discoverer) applyGatewayMetadata(fnInfo *FunctionInfo, fn *ast.FunctionNode, tc *typechecker.TypeChecker) {
	if fn == nil || len(fn.Params) != 1 || !fnInfo.IsResult {
		return
	}
	pt := d.resolveTypeName(fn.Params[0].GetType(), tc)
	st := fnInfo.ResultSuccessType
	if strings.Contains(pt, "GatewayRequest") && strings.Contains(st, "GatewayResponse") {
		fnInfo.IsGateway = true
	}
}

func (d *Discoverer) applyFunctionReturnMetadata(fnInfo *FunctionInfo, returnTypes []ast.TypeNode, tc *typechecker.TypeChecker) {
	if len(returnTypes) == 0 {
		d.log.Debugf("Function %s has no return types", fnInfo.Name)
		return
	}
	fnInfo.ReturnType = d.resolveTypeName(returnTypes[0], tc)
	fnInfo.ReturnTypes = make([]string, len(returnTypes))
	for i, rt := range returnTypes {
		fnInfo.ReturnTypes[i] = d.resolveTypeName(rt, tc)
	}
	fnInfo.HasMultipleReturns = len(returnTypes) > 1
	d.log.Debugf("Function %s has %d return types: %v, HasMultipleReturns: %v", fnInfo.Name, len(returnTypes), fnInfo.ReturnTypes, fnInfo.HasMultipleReturns)
	if len(returnTypes) == 1 {
		d.applyResultDiscoveryMetadata(fnInfo, returnTypes[0], tc)
	}
}
