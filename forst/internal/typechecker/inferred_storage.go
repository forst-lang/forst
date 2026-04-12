// Inferred type storage: hashes on AST nodes and function return signatures.
package typechecker

import (
	"fmt"
	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

// storeInferredType associates inferred types with an AST node using its structural hash.
func (tc *TypeChecker) storeInferredType(node ast.Node, types []ast.TypeNode) {
	// Ensure types have correct TypeKind
	processedTypes := make([]ast.TypeNode, len(types))
	for i, typ := range types {
		// For user-defined types, ensure they're marked as user-defined
		if !typ.IsHashBased() && !typ.IsGoBuiltin() {
			processedTypes[i] = ensureUserDefinedType(typ)
		} else {
			processedTypes[i] = typ
		}
	}

	if vn, ok := node.(ast.VariableNode); ok && vn.Ident.Span.IsSet() {
		k := variableOccurrenceKey{ident: vn.Ident.ID, span: vn.Ident.Span}
		tc.variableOccurrenceTypes[k] = processedTypes
	}
	hash, err := tc.Hasher.HashNode(node)
	if err != nil {
		tc.log.WithFields(logrus.Fields{
			"node":     node.String(),
			"function": "storeInferredType",
		}).WithError(err).Error("failed to hash node during storeInferredType")
		return
	}
	tc.Types[hash] = processedTypes
	tc.log.WithFields(logrus.Fields{
		"node":     node.String(),
		"key":      hash.ToTypeIdent(),
		"types":    processedTypes,
		"function": "storeInferredType",
		"hash":     fmt.Sprintf("%x", uint64(hash)),
	}).Trace("Stored inferred type for node")
}

// storeInferredFunctionReturnType stores the return types for a function in its signature.
func (tc *TypeChecker) storeInferredFunctionReturnType(fn *ast.FunctionNode, returnTypes []ast.TypeNode) {
	// Constructor-free Result returns: the body infers plain S but the function type stays Result(S,F).
	if len(fn.ReturnTypes) == 1 && len(returnTypes) == 1 &&
		fn.ReturnTypes[0].IsResultType() && !returnTypes[0].IsResultType() &&
		tc.isPlainSuccessCompatibleWithDeclaredResult(returnTypes[0], fn.ReturnTypes[0]) {
		returnTypes = []ast.TypeNode{fn.ReturnTypes[0]}
	}
	// Resolve aliased types for return types
	resolvedReturnTypes := make([]ast.TypeNode, len(returnTypes))
	for i, returnType := range returnTypes {
		resolvedType := tc.resolveAliasedType(returnType)

		// Ensure user-defined types have the correct TypeKind
		if resolvedType.TypeKind != ast.TypeKindHashBased && !tc.isBuiltinType(resolvedType.Ident) {
			resolvedReturnTypes[i] = ensureUserDefinedType(resolvedType)
		} else {
			resolvedReturnTypes[i] = resolvedType
		}
	}

	sig := tc.Functions[fn.Ident.ID]
	sig.ReturnTypes = resolvedReturnTypes
	tc.Functions[fn.Ident.ID] = sig
	tc.log.WithFields(logrus.Fields{
		"fn":          fn.Ident.ID,
		"returnTypes": resolvedReturnTypes,
		"sig":         sig,
		"function":    "storeInferredFunctionReturnType",
	}).Trace("Stored inferred function return type")
}
