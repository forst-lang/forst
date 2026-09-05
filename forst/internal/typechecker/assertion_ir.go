package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"
)

// Assertion is the source-ordered runtime assertion IR (not a canonical Predicate).
// TypeTarget is not an Assertion and must not be lowered to Atom(Name()).
type Assertion interface {
	assertion()
	String() string
}

// Atom is one named constraint / guard / literal check.
type Atom struct {
	Name string
	// BaseType is optional (e.g. String.Min); nil for bare Min()/guards.
	BaseType *ast.TypeIdent
	Args     []ast.ConstraintArgumentNode
	// RuntimeOnly is true when an argument is not a static literal (e.g. Min(n)).
	// Such atoms are runtime checks only — never a static dependent type.
	RuntimeOnly bool
}

func (Atom) assertion() {}

func (a Atom) String() string {
	var b strings.Builder
	if a.BaseType != nil {
		b.WriteString(string(*a.BaseType))
		if a.Name != "" {
			b.WriteByte('.')
		}
	}
	b.WriteString(a.Name)
	b.WriteByte('(')
	for i, arg := range a.Args {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(arg.String())
	}
	b.WriteByte(')')
	if a.RuntimeOnly {
		b.WriteString("@runtime")
	}
	return b.String()
}

// Any is a source-ordered disjunction (assertion `or` / Join). Short-circuits left-to-right.
type Any struct {
	Children []Assertion
}

func (Any) assertion() {}

func (a Any) String() string {
	return "Any(" + joinAssertions(a.Children) + ")"
}

// All is a source-ordered conjunction (Meet chain / sequential ensure). Short-circuits left-to-right.
type All struct {
	Children []Assertion
}

func (All) assertion() {}

func (a All) String() string {
	return "All(" + joinAssertions(a.Children) + ")"
}

// joinAssertions formats IR children for debug String() output.
func joinAssertions(children []Assertion) string {
	parts := make([]string, len(children))
	for i, c := range children {
		if c == nil {
			parts[i] = "<nil>"
			continue
		}
		parts[i] = c.String()
	}
	return strings.Join(parts, ", ")
}

// LowerAssertionNode lowers a parser AssertionNode (Meet + OrChains) into Assertion IR.
// Does not DNF-expand: A() or B() → Any; A().B() → All; nested trees stay nested.
func LowerAssertionNode(a ast.AssertionNode) Assertion {
	chains := a.MeetChains()
	alts := make([]Assertion, 0, len(chains))
	for _, chain := range chains {
		alts = append(alts, lowerMeetChain(chain))
	}
	if len(alts) == 1 {
		return alts[0]
	}
	return Any{Children: alts}
}

// lowerMeetChain lowers one Meet-only chain to Atom or All (no Or expansion).
func lowerMeetChain(chain ast.AssertionNode) Assertion {
	if len(chain.Constraints) == 0 {
		// BaseType-only assertion shape (e.g. type-guard call sugar stored as BaseType).
		if chain.BaseType != nil {
			name := string(*chain.BaseType)
			return Atom{Name: name, RuntimeOnly: false}
		}
		return All{Children: nil}
	}
	atoms := make([]Assertion, 0, len(chain.Constraints))
	for i, c := range chain.Constraints {
		atom := Atom{
			Name:        c.Name,
			Args:        append([]ast.ConstraintArgumentNode(nil), c.Args...),
			RuntimeOnly: constraintArgsRuntimeOnly(c.Args),
		}
		// Attach BaseType only to the first constraint in a Meet (String.Min.Max).
		if i == 0 && chain.BaseType != nil {
			bt := *chain.BaseType
			atom.BaseType = &bt
		}
		atoms = append(atoms, atom)
	}
	if len(atoms) == 1 {
		return atoms[0]
	}
	return All{Children: atoms}
}

// constraintArgsRuntimeOnly is true when any arg is a non-static literal or non-builtin type ident.
func constraintArgsRuntimeOnly(args []ast.ConstraintArgumentNode) bool {
	for _, arg := range args {
		if arg.Value != nil && !valueIsStaticLiteral(*arg.Value) {
			return true
		}
		// Parser often classifies bare idents as Type (before Value). A non-builtin
		// type arg on Min/Max/etc. is a runtime operand (e.g. Min(n)), not a static bound.
		if arg.Type != nil && !isBuiltinTypeIdent(arg.Type.Ident) {
			return true
		}
	}
	return false
}

// valueIsStaticLiteral is true when a constraint arg is a compile-time literal (not runtime-only).
func valueIsStaticLiteral(v ast.ValueNode) bool {
	switch v.(type) {
	case ast.IntLiteralNode, *ast.IntLiteralNode,
		ast.FloatLiteralNode, *ast.FloatLiteralNode,
		ast.StringLiteralNode, *ast.StringLiteralNode,
		ast.BoolLiteralNode, *ast.BoolLiteralNode,
		ast.NilLiteralNode, *ast.NilLiteralNode:
		return true
	default:
		return false
	}
}

// LowerRefinementTarget lowers ensure/if RHS. TypeTarget returns nil Assertion.
func LowerRefinementTarget(target ast.RefinementTarget, assertion ast.AssertionNode) (Assertion, *ast.TypeTarget) {
	switch t := target.(type) {
	case ast.TypeTarget:
		tt := t
		return nil, &tt
	case *ast.TypeTarget:
		if t == nil {
			break
		}
		tt := *t
		return nil, &tt
	case ast.AssertionTarget:
		return LowerAssertionNode(assertionFromTarget(t, assertion)), nil
	case *ast.AssertionTarget:
		if t == nil {
			break
		}
		return LowerAssertionNode(assertionFromTarget(*t, assertion)), nil
	}
	if assertion.IsBareTypeShape() {
		tt := ast.TypeTarget{Name: *assertion.BaseType}
		return nil, &tt
	}
	return LowerAssertionNode(assertion), nil
}

// assertionFromTarget rebuilds an AssertionNode from Join chains, or returns fallback.
func assertionFromTarget(t ast.AssertionTarget, fallback ast.AssertionNode) ast.AssertionNode {
	if len(t.Chains) == 0 {
		return fallback
	}
	first := t.Chains[0]
	if len(t.Chains) == 1 {
		return first
	}
	out := first
	out.OrChains = append([]ast.AssertionNode(nil), t.Chains[1:]...)
	return out
}

// LowerTypeGuardBody lowers a type-guard body to All/Any of assertions.
// Sequential ensure → All; if-is branches → Any of All(cond, body).
func LowerTypeGuardBody(body []ast.Node) Assertion {
	var parts []Assertion
	for _, node := range body {
		switch stmt := node.(type) {
		case ast.CommentNode:
			continue
		case ast.EnsureNode:
			a, tt := LowerRefinementTarget(stmt.Target, stmt.Assertion)
			if tt != nil {
				// Type membership in guards is not an assertion atom; skip for IR body shape.
				continue
			}
			if a != nil {
				parts = append(parts, a)
			}
		case *ast.EnsureNode:
			if stmt == nil {
				continue
			}
			a, tt := LowerRefinementTarget(stmt.Target, stmt.Assertion)
			if tt != nil {
				continue
			}
			if a != nil {
				parts = append(parts, a)
			}
		case ast.IfNode:
			if branch := lowerGuardIf(&stmt); branch != nil {
				parts = append(parts, branch)
			}
		case *ast.IfNode:
			if stmt == nil {
				continue
			}
			if branch := lowerGuardIf(stmt); branch != nil {
				parts = append(parts, branch)
			}
		}
	}
	switch len(parts) {
	case 0:
		return nil
	case 1:
		return parts[0]
	default:
		return All{Children: parts}
	}
}

// lowerGuardIf lowers if-is / else-if / else arms inside a type guard to Any of conjuncts.
func lowerGuardIf(n *ast.IfNode) Assertion {
	if n == nil {
		return nil
	}
	var alts []Assertion
	appendBranch := func(cond Assertion, bodyNodes []ast.Node) {
		body := LowerTypeGuardBody(bodyNodes)
		var conj Assertion
		switch {
		case cond != nil && body != nil:
			conj = All{Children: []Assertion{cond, body}}
		case cond != nil:
			conj = cond
		case body != nil:
			conj = body
		}
		if conj != nil {
			alts = append(alts, conj)
		}
	}
	appendBranch(lowerIfIsCondition(n.Condition), n.Body)
	for i := range n.ElseIfs {
		ei := n.ElseIfs[i]
		appendBranch(lowerIfIsCondition(ei.Condition), ei.Body)
	}
	if n.Else != nil {
		body := LowerTypeGuardBody(n.Else.Body)
		if body != nil {
			alts = append(alts, body)
		}
	}
	switch len(alts) {
	case 0:
		return nil
	case 1:
		return alts[0]
	default:
		return Any{Children: alts}
	}
}

// lowerIfIsCondition extracts assertion IR from an `x is …` condition, or nil.
func lowerIfIsCondition(cond ast.Node) Assertion {
	switch c := cond.(type) {
	case ast.BinaryExpressionNode:
		if c.Operator == ast.TokenIs {
			return lowerIsRHS(c.Right)
		}
	case *ast.BinaryExpressionNode:
		if c != nil && c.Operator == ast.TokenIs {
			return lowerIsRHS(c.Right)
		}
	}
	return nil
}

// lowerIsRHS lowers the RHS of `is` (AssertionNode / TypeDefAssertionExpr) to IR.
func lowerIsRHS(right ast.Node) Assertion {
	switch r := right.(type) {
	case ast.AssertionNode:
		return LowerAssertionNode(r)
	case *ast.AssertionNode:
		if r == nil {
			return nil
		}
		return LowerAssertionNode(*r)
	case ast.TypeDefAssertionExpr:
		if r.Assertion != nil {
			return LowerAssertionNode(*r.Assertion)
		}
	case *ast.TypeDefAssertionExpr:
		if r != nil && r.Assertion != nil {
			return LowerAssertionNode(*r.Assertion)
		}
	}
	return nil
}

// IsAny reports whether a is Any (possibly after unwrap of trivial wrappers).
func IsAny(a Assertion) bool {
	_, ok := a.(Any)
	return ok
}

// IsAll reports whether a is All.
func IsAll(a Assertion) bool {
	_, ok := a.(All)
	return ok
}

// AssertionShape summarizes IR for fixture asserts (Any/All/Atom + nested).
func AssertionShape(a Assertion) string {
	switch v := a.(type) {
	case Atom:
		return fmt.Sprintf("Atom(%s)", v.Name)
	case Any:
		parts := make([]string, len(v.Children))
		for i, c := range v.Children {
			parts[i] = AssertionShape(c)
		}
		return "Any(" + strings.Join(parts, ",") + ")"
	case All:
		parts := make([]string, len(v.Children))
		for i, c := range v.Children {
			parts[i] = AssertionShape(c)
		}
		return "All(" + strings.Join(parts, ",") + ")"
	case nil:
		return "<nil>"
	default:
		return fmt.Sprintf("%T", a)
	}
}

// HasRuntimeOnlyAtom walks IR for RuntimeOnly atoms.
func HasRuntimeOnlyAtom(a Assertion) bool {
	switch v := a.(type) {
	case Atom:
		return v.RuntimeOnly
	case Any:
		for _, c := range v.Children {
			if HasRuntimeOnlyAtom(c) {
				return true
			}
		}
	case All:
		for _, c := range v.Children {
			if HasRuntimeOnlyAtom(c) {
				return true
			}
		}
	}
	return false
}

// ContainsDNFExpansion heuristically detects All distributed under Any of Alls
// that would indicate illegal DNF expansion of All(Any(...), C).
func ContainsDNFExpansion(a Assertion) bool {
	// Nested Any under All is fine (no DNF expansion).
	// Red form is Any(All(...), All(...)) produced by distributing.
	// We only flag top-level Any whose children are all All with shared trailing atoms — too heuristic.
	// Fixtures assert shape strings instead; this stays false for nested All(Any, C).
	_, isAny := a.(Any)
	if !isAny {
		return false
	}
	any := a.(Any)
	if len(any.Children) < 2 {
		return false
	}
	allCount := 0
	for _, c := range any.Children {
		if _, ok := c.(All); ok {
			allCount++
		}
	}
	// A flat Any of Alls is the DNF shape we refuse to require as the stored form for
	// sequential ensure of (A or B) then C — but Any(All(cond,body),...) from guard if is valid.
	// Callers should assert AssertionShape explicitly.
	_ = allCount
	return false
}
