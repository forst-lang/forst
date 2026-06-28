package typechecker

import (
	"strings"
	"unicode"

	"forst/internal/ast"
)

func (tc *TypeChecker) seedKnownProviderRootsFromTypes() {
	eng := tc.providersEngine()
	for ident, def := range tc.Defs {
		typeDef, ok := def.(ast.TypeDefNode)
		if !ok {
			continue
		}
		fields := tc.typeDefMethodOnlyFields(typeDef)
		if fields == nil {
			continue
		}
		root := string(tc.providerRootIdent(ast.TypeNode{Ident: ident}))
		eng.KnownRoots[root] = ast.TypeNode{Ident: ident}
	}
}

func (tc *TypeChecker) typeDefMethodOnlyFields(typeDef ast.TypeDefNode) map[string]ast.ShapeFieldNode {
	if ae, ok := typeDefAssertionFromExpr(typeDef.Expr); ok && ae.Assertion != nil {
		fields := tc.resolveShapeFieldsFromAssertion(ae.Assertion)
		shape := ast.ShapeNode{Fields: fields}
		if shape.IsMethodOnlyContract() {
			return fields
		}
	}
	if se, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
		if se.Shape.IsMethodOnlyContract() {
			return se.Shape.Fields
		}
	}
	return nil
}

func (tc *TypeChecker) currentFunctionIdent() ast.Identifier {
	if tc.currentFunction == nil {
		return ""
	}
	return tc.currentFunction.Ident.ID
}

func (tc *TypeChecker) providerRootIdent(t ast.TypeNode) ast.Identifier {
	if t.Ident == ast.TypePointer && len(t.TypeParams) == 1 {
		return tc.providerRootIdent(t.TypeParams[0])
	}
	resolved := tc.resolveTypeAliasChain(t)
	return ast.Identifier(resolved.Ident)
}

// ProviderRootIdent returns the canonical root contract identifier for lowering.
func (tc *TypeChecker) ProviderRootIdent(t ast.TypeNode) ast.Identifier {
	return tc.providerRootIdent(t)
}

func (tc *TypeChecker) wiringValueAssignable(actual ast.TypeNode, contract ast.TypeNode) bool {
	if tc.IsTypeCompatible(actual, contract) {
		return true
	}
	if actual.Ident == ast.TypePointer && len(actual.TypeParams) == 1 {
		elem := actual.TypeParams[0]
		if tc.IsTypeCompatible(elem, contract) {
			return true
		}
		if def, ok := tc.Defs[contract.Ident]; ok {
			if expectedShape, ok := tc.getShapeFromTypeDef(def); ok && expectedShape.IsMethodOnlyContract() {
				return tc.typeMethodsSatisfyContract(elem.Ident, *expectedShape)
			}
		}
	}
	return false
}

func (tc *TypeChecker) resolveContractType(t ast.TypeNode) (ast.TypeNode, error) {
	if t.Assertion != nil {
		inferred, err := tc.InferAssertionType(t.Assertion, false, "", nil)
		if err != nil {
			return ast.TypeNode{}, err
		}
		if len(inferred) == 1 {
			return inferred[0], nil
		}
	}
	return t, nil
}

func (tc *TypeChecker) isExcludedProviderContract(t ast.TypeNode) bool {
	if t.Ident == ast.TypePointer && len(t.TypeParams) == 1 {
		return tc.isExcludedProviderContract(t.TypeParams[0])
	}
	key := tc.ProviderContractKey(t)
	return strings.HasSuffix(key, "testing.T") || key == "testing.T"
}

func (tc *TypeChecker) makeProviderSlot(contractType ast.TypeNode) (ProviderSlot, error) {
	resolved, err := tc.resolveContractType(contractType)
	if err != nil {
		return ProviderSlot{}, err
	}
	if tc.isExcludedProviderContract(resolved) {
		return ProviderSlot{}, nil
	}
	root := tc.providerRootIdent(resolved)
	key := tc.ProviderContractKey(resolved)
	slot := ProviderSlot{
		RootIdent:    root,
		ContractType: resolved,
		Key:          key,
	}
	tc.providersEngine().KnownRoots[string(root)] = resolved
	return slot, nil
}

func (tc *TypeChecker) recordDirectProvider(slot ProviderSlot) {
	if slot.Key == "" {
		return
	}
	fn := tc.currentFunctionIdent()
	if fn == "" {
		return
	}
	eng := tc.providersEngine()
	if eng.Direct[fn] == nil {
		eng.Direct[fn] = make(map[string]ProviderSlot)
	}
	eng.Direct[fn][slot.Key] = slot
}

func providerBindingName(root ast.Identifier) ast.Identifier {
	s := string(root)
	if idx := strings.LastIndex(s, "."); idx >= 0 {
		s = s[idx+1:]
	}
	if s == "" {
		return root
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return ast.Identifier(string(runes))
}

func (tc *TypeChecker) inferUseNode(use ast.UseNode) ([]ast.TypeNode, error) {
	slot, err := tc.makeProviderSlot(use.ContractType)
	if err != nil {
		return nil, err
	}
	if slot.Key != "" {
		tc.recordDirectProvider(slot)
	}

	bindName := use.Ident
	if bindName == nil {
		name := providerBindingName(slot.RootIdent)
		bindName = &ast.Ident{ID: name}
	}
	tc.scopeStack.currentScope().RegisterSymbol(
		bindName.ID,
		[]ast.TypeNode{slot.ContractType},
		SymbolVariable,
	)
	tc.VariableTypes[bindName.ID] = []ast.TypeNode{slot.ContractType}
	return nil, nil
}
