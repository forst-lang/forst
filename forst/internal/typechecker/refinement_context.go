package typechecker

// Fact is a proven refinement at a program point (phase 2d minimal).
type Fact struct {
	Subject   *AccessPath
	Predicate *Predicate
}

// RefinementContext owns program-point facts. Distinct from lexical Scope
// (which owns names/types/SymbolIDs).
type RefinementContext struct {
	// facts keyed by subject PathKey + predicate Key
	facts map[string]Fact
}

// NewRefinementContext creates an empty context.
func NewRefinementContext() *RefinementContext {
	return &RefinementContext{facts: make(map[string]Fact)}
}

// factKey combines subject PathKey and predicate Key for RefinementContext storage.
func factKey(subject *AccessPath, pred *Predicate) string {
	sk := ""
	if subject != nil {
		sk = subject.PathKey()
	}
	pk := ""
	if pred != nil {
		pk = pred.Key()
	}
	return sk + "\x00" + pk
}

// Clone returns a shallow copy of facts (copy-on-write friendly stub).
func (c *RefinementContext) Clone() *RefinementContext {
	if c == nil {
		return NewRefinementContext()
	}
	out := NewRefinementContext()
	for k, f := range c.facts {
		out.facts[k] = f
	}
	return out
}

// Prove records a proven predicate on subject.
func (c *RefinementContext) Prove(subject *AccessPath, pred *Predicate) {
	if c == nil || pred == nil {
		return
	}
	if c.facts == nil {
		c.facts = make(map[string]Fact)
	}
	c.facts[factKey(subject, pred)] = Fact{Subject: subject, Predicate: pred}
}

// Has reports whether subject has pred proven.
func (c *RefinementContext) Has(subject *AccessPath, pred *Predicate) bool {
	if c == nil || pred == nil {
		return false
	}
	_, ok := c.facts[factKey(subject, pred)]
	return ok
}

// Facts returns a snapshot of proven facts.
func (c *RefinementContext) Facts() []Fact {
	if c == nil {
		return nil
	}
	out := make([]Fact, 0, len(c.facts))
	for _, f := range c.facts {
		out = append(out, f)
	}
	return out
}

// Join intersects proven predicates (control-flow merge). Finite domains would
// union later; phase 2d only intersects exact predicate facts.
func JoinRefinementContexts(a, b *RefinementContext) *RefinementContext {
	if a == nil {
		return b.Clone()
	}
	if b == nil {
		return a.Clone()
	}
	out := NewRefinementContext()
	for k, f := range a.facts {
		if _, ok := b.facts[k]; ok {
			out.facts[k] = f
		}
	}
	return out
}

// TrueEdge returns the context after taking the true branch of an assertion
// (incoming plus proven predicate).
func (c *RefinementContext) TrueEdge(subject *AccessPath, pred *Predicate) *RefinementContext {
	out := c.Clone()
	out.Prove(subject, pred)
	return out
}

// FalseEdge returns the context after the false branch. Complements are deferred
// until finite domains (phase 3); for now the false edge keeps incoming facts only.
func (c *RefinementContext) FalseEdge() *RefinementContext {
	return c.Clone()
}

// LoopFixedPoint intersects entry with backedge (predicates ∩). Stub for phase 2d:
// one intersection step; callers may iterate until stable.
func LoopFixedPoint(entry, backedge *RefinementContext) *RefinementContext {
	return JoinRefinementContexts(entry, backedge)
}

// ControlEdgeKind distinguishes break / continue / return / fallthrough carriers (stubs).
type ControlEdgeKind uint8

const (
	EdgeBreak ControlEdgeKind = iota
	EdgeContinue
	EdgeReturn
	EdgeFallthrough
)

// ControlEdgeContext pairs a kind with a context snapshot.
type ControlEdgeContext struct {
	Kind ControlEdgeKind
	Ctx  *RefinementContext
}
