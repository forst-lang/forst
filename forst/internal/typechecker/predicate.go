package typechecker

import (
	"fmt"
	"sort"
	"strings"
)

// PredicateConnective is All or Any for canonical predicates (never DNF-distributed).
type PredicateConnective uint8

const (
	PredAtom PredicateConnective = iota // leaf Operand
	PredAll                             // conjunction (flattened, sorted)
	PredAny                             // disjunction (no DNF distribution)
)

// Operand is a leaf predicate operand (constraint name + optional static args key).
type Operand struct {
	Name        string
	ArgsKey     string
	RuntimeOnly bool
}

// Key returns a stable identity key for an Operand.
func (o Operand) Key() string {
	rt := ""
	if o.RuntimeOnly {
		rt = "@runtime"
	}
	return o.Name + "(" + o.ArgsKey + ")" + rt
}

// Predicate is an interned canonical refinement fact shape.
// Same connective is flattened; children are sorted by stable key and deduped.
// Never distribute Any over All (no DNF).
type Predicate struct {
	Conn     PredicateConnective
	Operand  *Operand     // when Conn == PredAtom
	Children []*Predicate // when Conn == PredAll or PredAny
	key      string
}

// Key returns the stable structural key (computed at intern time).
func (p *Predicate) Key() string {
	if p == nil {
		return ""
	}
	return p.key
}

// PredicateInterner canonicalizes Predicate values.
type PredicateInterner struct {
	byKey map[string]*Predicate
}

// NewPredicateInterner creates an empty interner.
func NewPredicateInterner() *PredicateInterner {
	return &PredicateInterner{byKey: make(map[string]*Predicate)}
}

// InternAtom interns a leaf operand.
func (pi *PredicateInterner) InternAtom(op Operand) *Predicate {
	return pi.intern(&Predicate{Conn: PredAtom, Operand: &op})
}

// InternAll interns an All of children (flatten / sort / dedupe).
func (pi *PredicateInterner) InternAll(children []*Predicate) *Predicate {
	return pi.internConnective(PredAll, children)
}

// InternAny interns an Any of children (flatten / sort / dedupe; no distribution).
func (pi *PredicateInterner) InternAny(children []*Predicate) *Predicate {
	return pi.internConnective(PredAny, children)
}

// FromAssertion builds a canonical Predicate from source-ordered Assertion IR.
// Nested structure is preserved (no DNF); only same-connective flatten/sort/dedupe applies.
func (pi *PredicateInterner) FromAssertion(a Assertion) *Predicate {
	switch v := a.(type) {
	case Atom:
		return pi.InternAtom(Operand{
			Name:        v.Name,
			ArgsKey:     atomArgsKey(v),
			RuntimeOnly: v.RuntimeOnly,
		})
	case All:
		kids := make([]*Predicate, 0, len(v.Children))
		for _, c := range v.Children {
			if p := pi.FromAssertion(c); p != nil {
				kids = append(kids, p)
			}
		}
		return pi.InternAll(kids)
	case Any:
		kids := make([]*Predicate, 0, len(v.Children))
		for _, c := range v.Children {
			if p := pi.FromAssertion(c); p != nil {
				kids = append(kids, p)
			}
		}
		return pi.InternAny(kids)
	default:
		return nil
	}
}

// atomArgsKey contributes constraint args to a Predicate interning key.
func atomArgsKey(a Atom) string {
	parts := make([]string, 0, len(a.Args)+1)
	if a.BaseType != nil {
		parts = append(parts, "base:"+string(*a.BaseType))
	}
	for _, arg := range a.Args {
		parts = append(parts, arg.String())
	}
	return strings.Join(parts, ",")
}

// internConnective builds a canonical And/Or Predicate (flatten, sort, dedupe).
func (pi *PredicateInterner) internConnective(conn PredicateConnective, children []*Predicate) *Predicate {
	flat := flattenPredChildren(conn, children)
	sort.SliceStable(flat, func(i, j int) bool {
		return flat[i].Key() < flat[j].Key()
	})
	flat = dedupePredChildren(flat)
	switch len(flat) {
	case 0:
		return pi.intern(&Predicate{Conn: conn})
	case 1:
		return flat[0]
	default:
		return pi.intern(&Predicate{Conn: conn, Children: flat})
	}
}

// flattenPredChildren merges nested same-connective children without DNF expansion.
func flattenPredChildren(conn PredicateConnective, children []*Predicate) []*Predicate {
	var out []*Predicate
	for _, c := range children {
		if c == nil {
			continue
		}
		if c.Conn == conn {
			out = append(out, flattenPredChildren(conn, c.Children)...)
			continue
		}
		out = append(out, c)
	}
	return out
}

// dedupePredChildren drops exact duplicate children by stable key.
func dedupePredChildren(children []*Predicate) []*Predicate {
	if len(children) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(children))
	out := make([]*Predicate, 0, len(children))
	for _, c := range children {
		k := c.Key()
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, c)
	}
	return out
}

// intern returns the shared canonical Predicate for p.
func (pi *PredicateInterner) intern(p *Predicate) *Predicate {
	if pi == nil {
		p.key = computePredicateKey(p)
		return p
	}
	if pi.byKey == nil {
		pi.byKey = make(map[string]*Predicate)
	}
	p.key = computePredicateKey(p)
	if existing, ok := pi.byKey[p.key]; ok {
		return existing
	}
	pi.byKey[p.key] = p
	return p
}

// computePredicateKey derives the structural interning key for a Predicate.
func computePredicateKey(p *Predicate) string {
	if p == nil {
		return ""
	}
	switch p.Conn {
	case PredAtom:
		if p.Operand == nil {
			return "atom:"
		}
		return "atom:" + p.Operand.Key()
	case PredAll:
		return "all:" + joinPredKeys(p.Children)
	case PredAny:
		return "any:" + joinPredKeys(p.Children)
	default:
		return fmt.Sprintf("conn:%d", p.Conn)
	}
}

// joinPredKeys concatenates child keys for a connective node.
func joinPredKeys(children []*Predicate) string {
	parts := make([]string, len(children))
	for i, c := range children {
		parts[i] = c.Key()
	}
	return strings.Join(parts, "|")
}

// Shape returns a debug shape string (All/Any/Atom) for tests.
func (p *Predicate) Shape() string {
	if p == nil {
		return "<nil>"
	}
	switch p.Conn {
	case PredAtom:
		if p.Operand == nil {
			return "Atom()"
		}
		return "Atom(" + p.Operand.Name + ")"
	case PredAll:
		parts := make([]string, len(p.Children))
		for i, c := range p.Children {
			parts[i] = c.Shape()
		}
		return "All(" + strings.Join(parts, ",") + ")"
	case PredAny:
		parts := make([]string, len(p.Children))
		for i, c := range p.Children {
			parts[i] = c.Shape()
		}
		return "Any(" + strings.Join(parts, ",") + ")"
	default:
		return "?"
	}
}
