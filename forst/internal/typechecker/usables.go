package typechecker

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// UsableSlot is one inferred Usable requirement for a function (root ident + contract type).
type UsableSlot struct {
	RootIdent    ast.Identifier
	ContractType ast.TypeNode
	Key          string
}

// Ambient holds merged Usable wiring keys in scope during a with-block.
type Ambient struct {
	keys     map[string]ast.TypeNode
	shadowed map[string]bool
}

type callSiteRecord struct {
	Callee      ast.Identifier
	AmbientKeys map[string]ast.TypeNode
	Span        ast.SourceSpan
}

type pendingWithCheck struct {
	with      ast.WithNode
	innerKeys map[string]struct{}
}

func newAmbient() Ambient {
	return Ambient{
		keys:     make(map[string]ast.TypeNode),
		shadowed: make(map[string]bool),
	}
}

func (tc *TypeChecker) initUsablesInference() {
	tc.FunctionUsables = make(map[ast.Identifier][]UsableSlot)
	tc.functionDirectUsables = make(map[ast.Identifier]map[string]UsableSlot)
	tc.functionCallSites = make(map[ast.Identifier][]callSiteRecord)
	tc.knownUsableRoots = make(map[string]ast.TypeNode)
	tc.ambientStack = nil
	tc.pendingWithChecks = nil
	tc.Warnings = nil
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

func (tc *TypeChecker) currentMergedAmbient() map[string]ast.TypeNode {
	if len(tc.ambientStack) == 0 {
		return nil
	}
	merged := make(map[string]ast.TypeNode)
	for i := range tc.ambientStack {
		a := tc.ambientStack[i]
		for k, v := range a.keys {
			merged[k] = v
		}
	}
	return merged
}

func (tc *TypeChecker) mergeAmbient(outer Ambient, inner Ambient) Ambient {
	out := newAmbient()
	for k, v := range outer.keys {
		if inner.shadowed[k] {
			continue
		}
		out.keys[k] = v
	}
	for k, v := range inner.keys {
		out.keys[k] = v
	}
	for k := range inner.shadowed {
		out.shadowed[k] = true
	}
	return out
}

func (tc *TypeChecker) ambientFromShape(shape ast.ShapeNode) (Ambient, error) {
	amb := newAmbient()
	for fieldName, field := range shape.Fields {
		if field.IsMethod {
			continue
		}
		fieldSpan := shapeFieldSpan(shape, fieldName, field)
		if err := tc.validateWiringKey(fieldName, fieldSpan); err != nil {
			return Ambient{}, err
		}
		expr, ok := field.ValueExpression()
		if !ok {
			return Ambient{}, fmt.Errorf("wiring field %s has no value", fieldName)
		}
		if _, isNil := expr.(ast.NilLiteralNode); isNil {
			return Ambient{}, diagnosticf(fieldSpan, "usables-nil-wiring",
				"wiring value for %s must not be nil", fieldName)
		}
		valTypes, err := tc.inferExpressionType(expr)
		if err != nil {
			return Ambient{}, err
		}
		if len(valTypes) != 1 {
			return Ambient{}, fmt.Errorf("wiring field %s must have a single type", fieldName)
		}
		contractType, ok := tc.knownUsableRoots[fieldName]
		if !ok {
			return Ambient{}, diagnosticf(fieldSpan, "usables-unknown-key",
				"unknown wiring key %q", fieldName)
		}
		if !tc.wiringValueAssignable(valTypes[0], contractType) {
			return Ambient{}, diagnosticf(fieldSpan, "usables-wiring-type",
				"wiring field %s: expected type %s, got %s", fieldName, contractType.Ident, valTypes[0].Ident)
		}
		amb.keys[fieldName] = contractType
		amb.shadowed[fieldName] = true
	}
	return amb, nil
}

func shapeFieldSpan(_ ast.ShapeNode, _ string, field ast.ShapeFieldNode) ast.SourceSpan {
	if field.Node != nil {
		if expr, ok := field.Node.(ast.ExpressionNode); ok {
			if s := spanOfExpression(expr); s.IsSet() {
				return s
			}
		}
	}
	return ast.SourceSpan{}
}

func (tc *TypeChecker) ambientFromInferredBundle(types []ast.TypeNode, span ast.SourceSpan) (Ambient, error) {
	if len(types) != 1 {
		return Ambient{}, fmt.Errorf("with wiring must have a single type")
	}
	shape, ok := tc.shapeFieldsForType(types[0])
	if !ok {
		return Ambient{}, diagnosticf(span, "usables-wiring-shape",
			"with wiring expression must be a Usables shape")
	}
	amb := newAmbient()
	for fieldName := range shape.Fields {
		if err := tc.validateWiringKey(fieldName, span); err != nil {
			return Ambient{}, err
		}
		contractType, ok := tc.knownUsableRoots[fieldName]
		if !ok {
			return Ambient{}, diagnosticf(span, "usables-unknown-key",
				"unknown wiring key %q", fieldName)
		}
		amb.keys[fieldName] = contractType
		amb.shadowed[fieldName] = true
	}
	return amb, nil
}

func (tc *TypeChecker) ambientFromWiringExpr(wiring ast.ExpressionNode, span ast.SourceSpan) (Ambient, error) {
	switch w := wiring.(type) {
	case ast.ShapeNode:
		return tc.ambientFromShape(w)
	default:
		types, err := tc.inferExpressionType(wiring)
		if err != nil {
			return Ambient{}, err
		}
		return tc.ambientFromInferredBundle(types, span)
	}
}

func (tc *TypeChecker) shapeFieldsForType(t ast.TypeNode) (ast.ShapeNode, bool) {
	if def, ok := tc.Defs[t.Ident]; ok {
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			if ae, ok := typeDefAssertionFromExpr(typeDef.Expr); ok && ae.Assertion != nil {
				fields := tc.resolveShapeFieldsFromAssertion(ae.Assertion)
				if len(fields) > 0 {
					return ast.ShapeNode{Fields: fields}, true
				}
			}
			if se, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
				return se.Shape, true
			}
		}
	}
	return ast.ShapeNode{}, false
}

func (tc *TypeChecker) validateWiringKey(key string, span ast.SourceSpan) error {
	ident := ast.TypeIdent(key)
	if def, ok := tc.Defs[ident]; ok {
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			if ae, ok := typeDefAssertionFromExpr(typeDef.Expr); ok && ae.Assertion != nil &&
				ae.Assertion.BaseType != nil && len(ae.Assertion.Constraints) == 0 {
				canonical := string(tc.usableRootIdent(ast.TypeNode{Ident: *ae.Assertion.BaseType}))
				if canonical != key {
					return diagnosticf(span, "usables-alias-key",
						"wiring key must be root contract ident %q, not alias %q", canonical, key)
				}
			}
		}
	}
	if _, ok := tc.knownUsableRoots[key]; ok {
		return nil
	}
	return diagnosticf(span, "usables-unknown-key", "unknown wiring key %q", key)
}

func (tc *TypeChecker) ambientSatisfiesSlot(slot UsableSlot, ambient map[string]ast.TypeNode) bool {
	if ambient == nil {
		return false
	}
	contract, ok := ambient[string(slot.RootIdent)]
	if !ok {
		return false
	}
	return tc.IsTypeCompatible(contract, slot.ContractType) || tc.IsTypeCompatible(slot.ContractType, contract)
}

func (tc *TypeChecker) checkCallUsablesSatisfied(callee ast.Identifier, ambient map[string]ast.TypeNode, span ast.SourceSpan) error {
	slots := tc.FunctionUsables[callee]
	var missing []string
	for _, slot := range slots {
		if !tc.ambientSatisfiesSlot(slot, ambient) {
			missing = append(missing, string(slot.RootIdent))
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return diagnosticf(span, "usables-unsatisfied",
		"%s requires %s; not supplied\n  required by: %s", callee, strings.Join(missing, ", "), callee)
}

func (tc *TypeChecker) recordFunctionCall(callee ast.Identifier, span ast.SourceSpan) {
	fn := tc.currentFunctionIdent()
	if fn == "" {
		return
	}
	rec := callSiteRecord{
		Callee:      callee,
		AmbientKeys: tc.currentMergedAmbient(),
		Span:        span,
	}
	tc.functionCallSites[fn] = append(tc.functionCallSites[fn], rec)
}

func (tc *TypeChecker) inferWithNode(with ast.WithNode) ([]ast.TypeNode, error) {
	inner, err := tc.ambientFromWiringExpr(with.Wiring, with.Span)
	if err != nil {
		return nil, err
	}

	var merged Ambient
	if len(tc.ambientStack) == 0 {
		merged = inner
	} else {
		outer := tc.ambientStack[len(tc.ambientStack)-1]
		merged = tc.mergeAmbient(outer, inner)
	}

	innerKeys := make(map[string]struct{}, len(inner.keys))
	for k := range inner.keys {
		innerKeys[k] = struct{}{}
	}
	tc.pendingWithChecks = append(tc.pendingWithChecks, pendingWithCheck{
		with:      with,
		innerKeys: innerKeys,
	})

	tc.ambientStack = append(tc.ambientStack, merged)
	tc.pushScope(with)

	for _, node := range with.Body {
		if _, err := tc.inferNodeType(node); err != nil {
			tc.popScope()
			tc.ambientStack = tc.ambientStack[:len(tc.ambientStack)-1]
			return nil, err
		}
	}

	tc.popScope()
	tc.ambientStack = tc.ambientStack[:len(tc.ambientStack)-1]
	return nil, nil
}

func (tc *TypeChecker) warnf(span ast.SourceSpan, code, format string, a ...interface{}) {
	tc.Warnings = append(tc.Warnings, Diagnostic{
		Msg:  fmt.Sprintf(format, a...),
		Span: span,
		Code: code,
	})
}

func (tc *TypeChecker) checkUnusedWiringKeys(check pendingWithCheck) {
	required := make(map[string]struct{})
	for _, node := range check.with.Body {
		tc.collectRequiredUsableRootsFromNode(node, required)
	}
	for key := range check.innerKeys {
		if _, needed := required[key]; !needed {
			tc.warnf(check.with.Span, "usables-unused-key",
				"wiring key %q is not required by any callee in this with-block", key)
		}
	}
}

func (tc *TypeChecker) collectRequiredUsableRootsFromNode(node ast.Node, out map[string]struct{}) {
	switch n := node.(type) {
	case ast.FunctionCallNode:
		callee := n.Function.ID
		for _, slot := range tc.FunctionUsables[callee] {
			out[string(slot.RootIdent)] = struct{}{}
		}
	case ast.WithNode:
		for _, child := range n.Body {
			tc.collectRequiredUsableRootsFromNode(child, out)
		}
	case ast.IfNode:
		for _, child := range n.Body {
			tc.collectRequiredUsableRootsFromNode(child, out)
		}
		if n.Else != nil {
			tc.collectRequiredUsableRootsFromNode(*n.Else, out)
		}
	case *ast.IfNode:
		tc.collectRequiredUsableRootsFromNode(*n, out)
	case ast.ForNode:
		for _, child := range n.Body {
			tc.collectRequiredUsableRootsFromNode(child, out)
		}
	case *ast.ForNode:
		for _, child := range n.Body {
			tc.collectRequiredUsableRootsFromNode(child, out)
		}
	case ast.EnsureNode:
		if n.Block != nil {
			for _, child := range n.Block.Body {
				tc.collectRequiredUsableRootsFromNode(child, out)
			}
		}
	}
}

func orderUsableSlots(slots []UsableSlot) []UsableSlot {
	if len(slots) <= 1 {
		return slots
	}
	seen := make(map[string]UsableSlot, len(slots))
	for _, s := range slots {
		seen[s.Key] = s
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]UsableSlot, 0, len(keys))
	for _, k := range keys {
		out = append(out, seen[k])
	}
	return out
}

func (tc *TypeChecker) slotsFromMap(m map[string]UsableSlot) []UsableSlot {
	if len(m) == 0 {
		return nil
	}
	out := make([]UsableSlot, 0, len(m))
	for _, slot := range m {
		out = append(out, slot)
	}
	return orderUsableSlots(out)
}

func (tc *TypeChecker) addUsableSlotToFunction(fn ast.Identifier, slot UsableSlot) bool {
	if slot.Key == "" {
		return false
	}
	existing := tc.FunctionUsables[fn]
	for _, s := range existing {
		if s.Key == slot.Key {
			return false
		}
	}
	tc.FunctionUsables[fn] = append(existing, slot)
	return true
}

func (tc *TypeChecker) computeUsablesFixedPoint() {
	for fn, direct := range tc.functionDirectUsables {
		tc.FunctionUsables[fn] = orderUsableSlots(tc.slotsFromMap(direct))
	}

	changed := true
	for changed {
		changed = false
		for fn, sites := range tc.functionCallSites {
			for _, site := range sites {
				for _, slot := range tc.FunctionUsables[site.Callee] {
					if tc.ambientSatisfiesSlot(slot, site.AmbientKeys) {
						continue
					}
					if tc.addUsableSlotToFunction(fn, slot) {
						changed = true
					}
				}
			}
		}
	}

	for fn, slots := range tc.FunctionUsables {
		tc.FunctionUsables[fn] = orderUsableSlots(slots)
	}

	tc.log.WithFields(logrus.Fields{
		"function": "computeUsablesFixedPoint",
		"count":    len(tc.FunctionUsables),
	}).Debug("Computed FunctionUsables fixed point")
}

func (tc *TypeChecker) callerForwardsUsables(caller, callee ast.Identifier) bool {
	calleeSlots := tc.FunctionUsables[callee]
	if len(calleeSlots) == 0 {
		return true
	}
	callerKeys := make(map[string]struct{})
	for _, slot := range tc.FunctionUsables[caller] {
		callerKeys[slot.Key] = struct{}{}
	}
	for _, slot := range calleeSlots {
		if _, ok := callerKeys[slot.Key]; !ok {
			return false
		}
	}
	return true
}

func (tc *TypeChecker) isWiringRootIdent(id ast.Identifier) bool {
	return ast.IsUsablesWiringRoot(id, tc.paramTypesForIdent(id))
}

func (tc *TypeChecker) paramTypesForIdent(id ast.Identifier) []ast.TypeNode {
	sig, ok := tc.Functions[id]
	if !ok {
		return nil
	}
	types := make([]ast.TypeNode, 0, len(sig.Parameters))
	for _, param := range sig.Parameters {
		types = append(types, param.Type)
	}
	return types
}

func (tc *TypeChecker) validateCallSite(caller ast.Identifier, site callSiteRecord) error {
	if len(tc.FunctionUsables[site.Callee]) == 0 {
		return nil
	}
	if tc.ambientSatisfiesAllSlots(site.AmbientKeys, tc.FunctionUsables[site.Callee]) {
		return nil
	}
	if !tc.isWiringRootIdent(caller) && tc.callerForwardsUsables(caller, site.Callee) {
		return nil
	}
	return tc.checkCallUsablesSatisfied(site.Callee, site.AmbientKeys, site.Span)
}

func (tc *TypeChecker) ambientSatisfiesAllSlots(ambient map[string]ast.TypeNode, slots []UsableSlot) bool {
	for _, slot := range slots {
		if !tc.ambientSatisfiesSlot(slot, ambient) {
			return false
		}
	}
	return true
}

func (tc *TypeChecker) validateAllCallSites() error {
	for caller, sites := range tc.functionCallSites {
		for _, site := range sites {
			if err := tc.validateCallSite(caller, site); err != nil {
				return err
			}
		}
	}
	return nil
}

func (tc *TypeChecker) isWiringRoot(fn ast.FunctionNode) bool {
	return ast.IsUsablesWiringRoot(fn.Ident.ID, ast.ParamTypesFromFunction(fn))
}

func obligationChain(root ast.Identifier, slots []UsableSlot) string {
	parts := []string{string(root)}
	seen := map[string]struct{}{string(root): {}}
	for _, slot := range slots {
		name := string(slot.RootIdent)
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		parts = append(parts, name)
	}
	return strings.Join(parts, " → ")
}

func (tc *TypeChecker) finishUsablesChecking(nodes []ast.Node) error {
	tc.computeUsablesFixedPoint()

	if err := tc.validateAllCallSites(); err != nil {
		return err
	}

	for _, check := range tc.pendingWithChecks {
		tc.checkUnusedWiringKeys(check)
	}

	for _, node := range nodes {
		fn, ok := node.(ast.FunctionNode)
		if !ok {
			continue
		}
		if !tc.isWiringRoot(fn) {
			continue
		}
		slots := tc.FunctionUsables[fn.Ident.ID]
		if len(slots) == 0 {
			continue
		}
		names := make([]string, len(slots))
		for i, s := range slots {
			names[i] = string(s.RootIdent)
		}
		chain := obligationChain(fn.Ident.ID, slots)
		return diagnosticf(fn.Ident.Span, "usables-unsatisfied",
			"%s requires %s; not supplied at wiring root\n  required by: %s",
			fn.Ident.ID, strings.Join(names, ", "), chain)
	}
	return nil
}
