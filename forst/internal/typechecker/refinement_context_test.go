package typechecker

import "testing"

func TestRefinementContext_trueFalseEdgesAndJoin(t *testing.T) {
	pi := NewPredicateInterner()
	path := NewPathInterner().Intern(AccessPath{Root: 1})
	predA := pi.InternAtom(Operand{Name: "A"})
	predB := pi.InternAtom(Operand{Name: "B"})

	base := NewRefinementContext()
	trueA := base.TrueEdge(path, predA)
	trueB := base.TrueEdge(path, predB)
	if !trueA.Has(path, predA) || trueA.Has(path, predB) {
		t.Fatal("true edge A")
	}
	falseA := trueA.FalseEdge()
	if !falseA.Has(path, predA) {
		t.Fatal("false edge keeps incoming (no complement yet)")
	}

	joined := JoinRefinementContexts(trueA, trueB)
	if joined.Has(path, predA) || joined.Has(path, predB) {
		t.Fatal("join must intersect; A∩B is empty")
	}

	both := trueA.Clone()
	both.Prove(path, predB)
	joined2 := JoinRefinementContexts(both, trueA)
	if !joined2.Has(path, predA) || joined2.Has(path, predB) {
		t.Fatal("join intersects to A only")
	}
}

func TestRefinementContext_loopFixedPoint(t *testing.T) {
	pi := NewPredicateInterner()
	path := NewPathInterner().Intern(AccessPath{Root: 2})
	pred := pi.InternAtom(Operand{Name: "Stable"})
	entry := NewRefinementContext()
	entry.Prove(path, pred)
	back := NewRefinementContext() // zero-iteration / dropped fact
	fp := LoopFixedPoint(entry, back)
	if fp.Has(path, pred) {
		t.Fatal("fixed point intersects away facts missing on backedge")
	}
}
