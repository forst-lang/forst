package typechecker

import (
	"strings"
	"unicode"

	"forst/internal/ast"
)

func (tc *TypeChecker) initUsablesInference() {
	tc.FunctionUsables = make(map[ast.Identifier][]UsableSlot)
	tc.functionDirectUsables = make(map[ast.Identifier]map[string]UsableSlot)
	tc.functionCallSites = make(map[ast.Identifier][]callSiteRecord)
	tc.knownUsableRoots = make(map[string]ast.TypeNode)
	tc.ambientStack = nil
	tc.pendingWithChecks = nil
	tc.Warnings = nil
	tc.crossPackageCallSites = nil
}

func (tc *TypeChecker) seedKnownUsableRootsFromTypes() {
	for ident, def := range tc.Defs {
		typeDef, ok := def.(ast.TypeDefNode)
		if !ok {
			continue
		}
		fields := tc.typeDefMethodOnlyFields(typeDef)
		if fields == nil {
			continue
		}
		root := string(tc.usableRootIdent(ast.TypeNode{Ident: ident}))
		tc.knownUsableRoots[root] = ast.TypeNode{Ident: ident}
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

func (tc *TypeChecker) usableRootIdent(t ast.TypeNode) ast.Identifier {
	if t.Ident == ast.TypePointer && len(t.TypeParams) == 1 {
		return tc.usableRootIdent(t.TypeParams[0])
	}
	resolved := tc.resolveTypeAliasChain(t)
	return ast.Identifier(resolved.Ident)
}

// UsableRootIdent returns the canonical root contract identifier for lowering.
func (tc *TypeChecker) UsableRootIdent(t ast.TypeNode) ast.Identifier {
	return tc.usableRootIdent(t)
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

func (tc *TypeChecker) isExcludedUsableContract(t ast.TypeNode) bool {
	if t.Ident == ast.TypePointer && len(t.TypeParams) == 1 {
		return tc.isExcludedUsableContract(t.TypeParams[0])
	}
	key := tc.UsableContractKey(t)
	return strings.HasSuffix(key, "testing.T") || key == "testing.T"
}

func (tc *TypeChecker) makeUsableSlot(contractType ast.TypeNode) (UsableSlot, error) {
	resolved, err := tc.resolveContractType(contractType)
	if err != nil {
		return UsableSlot{}, err
	}
	if tc.isExcludedUsableContract(resolved) {
		return UsableSlot{}, nil
	}
	root := tc.usableRootIdent(resolved)
	key := tc.UsableContractKey(resolved)
	slot := UsableSlot{
		RootIdent:    root,
		ContractType: resolved,
		Key:          key,
	}
	tc.knownUsableRoots[string(root)] = resolved
	return slot, nil
}

func (tc *TypeChecker) recordDirectUsable(slot UsableSlot) {
	if slot.Key == "" {
		return
	}
	fn := tc.currentFunctionIdent()
	if fn == "" {
		return
	}
	if tc.functionDirectUsables[fn] == nil {
		tc.functionDirectUsables[fn] = make(map[string]UsableSlot)
	}
	tc.functionDirectUsables[fn][slot.Key] = slot
}

func usableBindingName(root ast.Identifier) ast.Identifier {
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
	slot, err := tc.makeUsableSlot(use.ContractType)
	if err != nil {
		return nil, err
	}
	if slot.Key != "" {
		tc.recordDirectUsable(slot)
	}

	bindName := use.Ident
	if bindName == nil {
		name := usableBindingName(slot.RootIdent)
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
