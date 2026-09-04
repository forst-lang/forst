package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"
)

// requiredFactForCallee returns the named refinement fact a call requires on its
// first argument, when the callee is a fact-use helper from the refinement suite
// (acceptAdult, needA, usePresent, …). Empty string means no fact requirement.
func requiredFactForCallee(fn ast.Identifier) string {
	switch string(fn) {
	case "acceptAdult":
		return "Adult"
	case "ship":
		return "Shippable"
	case "useValid":
		return "ValidPeriod"
	case "needA":
		return "A"
	case "usePresent", "useSession", "useUser":
		return "Present"
	case "needAdultOrAdmin":
		return "Adult|Admin"
	case "useAllowed":
		return "AllowedFor"
	case "useHasUserEmail":
		return "HasUserEmail"
	case "pay":
		return "Payable"
	case "acceptAdultUsers", "needAllAdults":
		return "AllAdults"
	case "needValid", "needValidUser":
		return "ValidUser"
	case "needTagged":
		return "Tagged"
	case "needPriced":
		return "Priced"
	case "needValidMap":
		return "ValidMap"
	case "useAdultAge", "needMin1":
		return "Min"
	default:
		return ""
	}
}

// checkCallFactRequirements errors when a fact-use callee's first arg lacks the required refinement.
func (tc *TypeChecker) checkCallFactRequirements(fn ast.Identifier, e ast.FunctionCallNode) error {
	req := requiredFactForCallee(fn)
	if req == "" || len(e.Arguments) == 0 {
		return nil
	}
	arg := e.Arguments[0]
	sp := spanForCallArg(e.ArgSpans, 0, e.Arguments, e.CallSpan)
	if !sp.IsSet() {
		sp = e.Function.Span
	}
	if req == "Adult|Admin" {
		if tc.exprHasGuardFact(arg, "Adult") || tc.exprHasGuardFact(arg, "Admin") {
			// Compound or fact: only the compound name counts, not salvaged disjuncts.
			if tc.exprHasGuardFact(arg, "Adult") && !tc.exprHasCompoundOrFact(arg) {
				return diagnosticf(sp, "refinement-missing-fact",
					"refinement-missing-fact: %s requires Adult|Admin on argument, not a salvaged disjunct", fn)
			}
		}
		if tc.exprHasCompoundOrFact(arg) {
			return nil
		}
		return diagnosticf(sp, "refinement-missing-fact",
			"refinement-missing-fact: call %s requires fact Adult|Admin on argument", fn)
	}
	if tc.exprHasGuardFact(arg, req) {
		return nil
	}
	argPath := tc.accessPathForExpr(arg)
	if d := tc.findDroppedFact(req, argPath); d != nil {
		code := diagnosticCodeForDrop(d.Reason)
		switch d.Reason {
		case dropByForeign:
			return diagnosticf(sp, code,
				"%s: fact %s was invalidated by an untrusted Go/reflect/unsafe boundary; Forst cannot prove the value unchanged — recover with ensure", code, req)
		case dropByConcurrent:
			return diagnosticf(sp, code,
				"%s: fact %s was invalidated by concurrent escape (go/channel/closure); recover with ensure", code, req)
		case dropByAlias:
			return diagnosticf(sp, code,
				"%s: fact %s was invalidated by a write through an alias; recover with ensure", code, req)
		case dropByCall:
			return diagnosticf(sp, code,
				"%s: fact %s was invalidated by a call; recover with ensure", code, req)
		default:
			return diagnosticf(sp, code,
				"%s: fact %s was established earlier but invalidated by a write; recover with ensure", code, req)
		}
	}
	return diagnosticf(sp, "refinement-missing-fact",
		"refinement-missing-fact: call %s requires fact %s on argument", fn, req)
}

// exprHasCompoundOrFact reports a proven Join fact (Adult|Admin / Any(…)) on arg.
func (tc *TypeChecker) exprHasCompoundOrFact(arg ast.ExpressionNode) bool {
	guards := tc.narrowingGuardsForExpr(arg)
	for _, g := range guards {
		if strings.Contains(g, "|") || strings.HasPrefix(g, "Any(") {
			return true
		}
	}
	return false
}

// exprHasGuardFact reports whether arg currently carries the named narrowing / refinement fact.
func (tc *TypeChecker) exprHasGuardFact(arg ast.ExpressionNode, guard string) bool {
	if tc == nil || guard == "" {
		return false
	}
	for _, g := range tc.narrowingGuardsForExpr(arg) {
		if g == guard {
			return true
		}
	}
	// RefinementContext lookup by access path.
	path := tc.accessPathForExpr(arg)
	if path != nil && tc.predicates != nil {
		pred := tc.predicates.InternAtom(Operand{Name: guard})
		if tc.CurrentRefinementContext().Has(path, pred) {
			return true
		}
	}
	return false
}

// narrowingGuardsForExpr collects guard names from occurrence maps, symbols, and compound narrowing.
func (tc *TypeChecker) narrowingGuardsForExpr(arg ast.ExpressionNode) []string {
	if tc == nil || arg == nil {
		return nil
	}
	if vn, ok := arg.(ast.VariableNode); ok {
		if g := tc.NarrowingTypeGuardsForVariableOccurrence(vn); len(g) > 0 {
			return g
		}
		if sym, ok := tc.CurrentScope().LookupVariable(vn.Ident.ID); ok {
			return append([]string(nil), sym.NarrowingTypeGuards...)
		}
		if info, ok := tc.compoundNarrowingByIdentifier[vn.Ident.ID]; ok {
			return append([]string(nil), info.guards...)
		}
		return nil
	}
	if id := dottedIdentFromExpr(arg); id != "" {
		fake := ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(id)}}
		if g := tc.NarrowingTypeGuardsForVariableOccurrence(fake); len(g) > 0 {
			return g
		}
		if sym, ok := tc.CurrentScope().LookupVariable(ast.Identifier(id)); ok {
			return append([]string(nil), sym.NarrowingTypeGuards...)
		}
		if info, ok := tc.compoundNarrowingByIdentifier[ast.Identifier(id)]; ok {
			return append([]string(nil), info.guards...)
		}
	}
	return nil
}

// accessPathForExpr builds an AccessPath for a variable or dotted field chain.
func (tc *TypeChecker) accessPathForExpr(arg ast.ExpressionNode) *AccessPath {
	if vn, ok := arg.(ast.VariableNode); ok {
		return tc.AccessPathForVariable(&vn)
	}
	if id := dottedIdentFromExpr(arg); id != "" {
		vn := ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(id)}}
		return tc.AccessPathForVariable(&vn)
	}
	return nil
}

// dottedIdentFromExpr flattens Variable / FieldAccess chains to "a.b.c".
func dottedIdentFromExpr(expr ast.ExpressionNode) string {
	switch e := expr.(type) {
	case ast.VariableNode:
		return string(e.Ident.ID)
	case *ast.VariableNode:
		if e == nil {
			return ""
		}
		return string(e.Ident.ID)
	case ast.FieldAccessNode:
		base := dottedIdentFromExpr(e.Target)
		field := string(e.Field.ID)
		if base == "" {
			return field
		}
		return base + "." + field
	case *ast.FieldAccessNode:
		if e == nil {
			return ""
		}
		base := dottedIdentFromExpr(e.Target)
		field := string(e.Field.ID)
		if base == "" {
			return field
		}
		return base + "." + field
	default:
		return ""
	}
}

// proveAssertionOnSubject records the proven predicate on subject in RefinementContext.
func (tc *TypeChecker) proveAssertionOnSubject(subject ast.VariableNode, a *ast.AssertionNode) {
	if tc == nil || a == nil || tc.predicates == nil {
		return
	}
	ir := LowerAssertionNode(*a)
	if ir == nil {
		return
	}
	path := tc.AccessPathForVariable(&subject)
	tc.CurrentRefinementContext().Prove(path, tc.predicates.FromAssertion(ir))
}

// missingFactError formats a stable diagnostic (kept for callers that want fmt.Errorf).
func missingFactError(fn, guard string) error {
	return fmt.Errorf("refinement-missing-fact: call %s requires fact %s on argument", fn, guard)
}
