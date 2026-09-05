package typechecker

import (
	"strings"

	"forst/internal/ast"
)

// ensureIRRecord stores lowered assertion IR for one ensure / if-is site.
type ensureIRRecord struct {
	Assertion Assertion
	TypeTarget *ast.TypeTarget // non-nil when RHS is a type target (not an Atom)
}

// recordEnsureIR lowers and stores IR for an ensure statement.
func (tc *TypeChecker) recordEnsureIR(n ast.EnsureNode) {
	if tc == nil {
		return
	}
	if tc.ensureIR == nil {
		tc.ensureIR = make(map[string]ensureIRRecord)
	}
	a, tt := LowerRefinementTarget(n.Target, n.Assertion)
	key := ensureIRKey(n)
	tc.ensureIR[key] = ensureIRRecord{Assertion: a, TypeTarget: tt}
	tc.lastEnsureIR = ensureIRRecord{Assertion: a, TypeTarget: tt}
	if a != nil && tc.predicates != nil {
		_ = tc.predicates.FromAssertion(a)
	}
}

// recordIfIsIR lowers and stores IR for an if-is condition (same Any as ensure).
func (tc *TypeChecker) recordIfIsIR(condition ast.Node) {
	if tc == nil || condition == nil {
		return
	}
	bin, ok := condition.(ast.BinaryExpressionNode)
	if !ok || bin.Operator != ast.TokenIs {
		return
	}
	a := lowerIsRHS(bin.Right)
	if a == nil {
		return
	}
	if tc.ifIsIR == nil {
		tc.ifIsIR = make([]Assertion, 0, 4)
	}
	tc.ifIsIR = append(tc.ifIsIR, a)
	tc.lastIfIsIR = a
	if tc.predicates != nil {
		_ = tc.predicates.FromAssertion(a)
	}
}

// recordTypeGuardBodyIR stores All/Any IR for a type guard body.
func (tc *TypeChecker) recordTypeGuardBodyIR(name ast.Identifier, body []ast.Node) {
	if tc == nil {
		return
	}
	ir := LowerTypeGuardBody(body)
	if tc.guardBodyIR == nil {
		tc.guardBodyIR = make(map[ast.Identifier]Assertion)
	}
	tc.guardBodyIR[name] = ir
	tc.lastGuardBodyIR = ir
	if tc.predicates != nil {
		_ = tc.predicates.FromAssertion(ir)
	}
}

// ensureIRKey is a stable map key for one ensure site (subject + assertion text).
func ensureIRKey(n ast.EnsureNode) string {
	return string(n.Variable.Ident.ID) + "\x00" + n.Assertion.String()
}

// EnsureAssertionIR returns the lowered IR for the most recent ensure (tests).
func (tc *TypeChecker) EnsureAssertionIR() Assertion {
	if tc == nil {
		return nil
	}
	return tc.lastEnsureIR.Assertion
}

// EnsureTypeTarget returns the TypeTarget for the most recent ensure, if any.
func (tc *TypeChecker) EnsureTypeTarget() *ast.TypeTarget {
	if tc == nil {
		return nil
	}
	return tc.lastEnsureIR.TypeTarget
}

// GuardBodyIR returns lowered IR for a type guard by name.
func (tc *TypeChecker) GuardBodyIR(name ast.Identifier) Assertion {
	if tc == nil || tc.guardBodyIR == nil {
		return nil
	}
	return tc.guardBodyIR[name]
}

// LastGuardBodyIR returns the most recently recorded guard body IR.
func (tc *TypeChecker) LastGuardBodyIR() Assertion {
	if tc == nil {
		return nil
	}
	return tc.lastGuardBodyIR
}

// LastIfIsIR returns the most recently recorded if-is assertion IR.
func (tc *TypeChecker) LastIfIsIR() Assertion {
	if tc == nil {
		return nil
	}
	return tc.lastIfIsIR
}

// CollectEnsureAssertionIRs returns all ensure IRs recorded this pass (assertion sites only).
func (tc *TypeChecker) CollectEnsureAssertionIRs() []Assertion {
	if tc == nil || tc.ensureIR == nil {
		return nil
	}
	out := make([]Assertion, 0, len(tc.ensureIR))
	for _, rec := range tc.ensureIR {
		if rec.Assertion != nil {
			out = append(out, rec.Assertion)
		}
	}
	return out
}

// AccessPathForVariable builds and interns an AccessPath for a variable / field path subject.
func (tc *TypeChecker) AccessPathForVariable(v *ast.VariableNode) *AccessPath {
	if tc == nil || v == nil {
		return nil
	}
	name := string(v.Ident.ID)
	rootName := name
	var steps []AccessStep
	if i := strings.IndexByte(name, '.'); i >= 0 {
		rootName = name[:i]
		rest := name[i+1:]
		for _, part := range strings.Split(rest, ".") {
			if part == "*" || part == "[*]" {
				steps = append(steps, AccessStep{Kind: AccessIndexAny})
				continue
			}
			steps = append(steps, AccessStep{Kind: AccessField, Field: part})
		}
	}
	id, ok := tc.CurrentScope().LookupSymbolID(ast.Identifier(rootName))
	if !ok {
		// Allocate ephemeral path root for unresolved names (tests / early lower).
		id = 0
	}
	p := AccessPath{Root: id, Steps: steps}
	if tc.paths == nil {
		tc.paths = NewPathInterner()
	}
	return tc.paths.Intern(p)
}

// CurrentRefinementContext returns the active refinement context (creates if needed).
func (tc *TypeChecker) CurrentRefinementContext() *RefinementContext {
	if tc == nil {
		return nil
	}
	if tc.refinementCtx == nil {
		tc.refinementCtx = NewRefinementContext()
	}
	return tc.refinementCtx
}
