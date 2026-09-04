package typechecker

import (
	"forst/internal/ast"
)

// dropReason distinguishes invalidation diagnostics (phase 4b–4g).
type dropReason string

const (
	dropByWrite      dropReason = "write"
	dropByAlias      dropReason = "alias"
	dropByCall       dropReason = "call"
	dropByForeign    dropReason = "foreign"
	dropByConcurrent dropReason = "concurrent"
)

// droppedFact records a fact removed by a write (for use-site diagnostics).
type droppedFact struct {
	PredicateName string
	SubjectKey    string
	EstablishedAt ast.SourceSpan
	InvalidatedBy ast.SourceSpan
	Reason        dropReason
}

// invalidateOverlappingFacts drops facts clobbered by writePath (reason: write).
func (tc *TypeChecker) invalidateOverlappingFacts(writePath *AccessPath, writeSpan ast.SourceSpan) {
	tc.invalidateOverlappingFactsWithReason(writePath, writeSpan, dropByWrite)
}

// invalidateOverlappingFactsWithReason expands aliases then drops MayClobber-overlapping facts.
func (tc *TypeChecker) invalidateOverlappingFactsWithReason(writePath *AccessPath, writeSpan ast.SourceSpan, reason dropReason) {
	if tc == nil || writePath == nil {
		return
	}
	if reason == "" {
		reason = dropByWrite
	}
	paths := tc.expandWriteThroughAliases(writePath)
	seen := map[string]bool{}
	for _, p := range paths {
		if p == nil {
			continue
		}
		key := p.PathKey()
		if seen[key] {
			continue
		}
		seen[key] = true
		r := reason
		if p != writePath && reason == dropByWrite {
			r = dropByAlias
		}
		tc.invalidateOneWritePath(p, writeSpan, r)
	}
}

// invalidateOneWritePath filters refinementFacts for one concrete write path and records drops.
func (tc *TypeChecker) invalidateOneWritePath(writePath *AccessPath, writeSpan ast.SourceSpan, reason dropReason) {
	var kept []RefinementFact
	for _, f := range tc.refinementFacts {
		drop := false
		for _, r := range f.Reads {
			if MayClobber(writePath, r) {
				drop = true
				break
			}
		}
		// Whole-subject rebind (`user = other`): write path is exactly the subject root.
		if !drop && f.Subject != nil && len(writePath.Steps) == 0 && writePath.Root == f.Subject.Root {
			drop = true
		}
		// Collection / subject-root: any write under the subject root drops the fact
		// when the fact's reads include IndexAny (coarse collection membership).
		if !drop && f.Subject != nil && writePath.Root == f.Subject.Root {
			if readsContainIndexAny(f.Reads) || writeContainsIndexAny(writePath) {
				drop = true
			}
		}
		if drop {
			name := ""
			if f.Predicate != nil && f.Predicate.Operand != nil {
				name = f.Predicate.Operand.Name
			} else if f.Predicate != nil {
				name = f.Predicate.Shape()
			}
			tc.droppedFacts = append(tc.droppedFacts, droppedFact{
				PredicateName: name,
				SubjectKey:    pathKeyOrEmpty(f.Subject),
				EstablishedAt: f.EstablishedAt,
				InvalidatedBy: writeSpan,
				Reason:        reason,
			})
			tc.clearNarrowingForFact(f)
			continue
		}
		kept = append(kept, f)
	}
	tc.refinementFacts = kept
}

// readsContainIndexAny is true if any dependency uses a coarse [*] element path.
func readsContainIndexAny(reads AccessPaths) bool {
	for i := range reads {
		if writeContainsIndexAny(reads[i]) {
			return true
		}
	}
	return false
}

// writeContainsIndexAny is true if the write path includes an IndexAny step.
func writeContainsIndexAny(p *AccessPath) bool {
	if p == nil {
		return false
	}
	for _, s := range p.Steps {
		if s.Kind == AccessIndexAny {
			return true
		}
	}
	return false
}

// pathKeyOrEmpty returns PathKey or "" for nil paths.
func pathKeyOrEmpty(p *AccessPath) string {
	if p == nil {
		return ""
	}
	return p.PathKey()
}

// clearNarrowingForFact removes the dropped predicate from scopes, compound maps, and RefinementContext.
func (tc *TypeChecker) clearNarrowingForFact(f RefinementFact) {
	if tc == nil || f.Predicate == nil {
		return
	}
	name := ""
	if f.Predicate.Operand != nil {
		name = f.Predicate.Operand.Name
	}
	if name == "" {
		return
	}

	tc.clearGuardNameFromScope(tc.CurrentScope(), name)
	tc.clearCompoundNarrowing(name)
	tc.clearRefinementContext(f)
}

// clearCompoundNarrowing strips guard name from compoundNarrowingByIdentifier entries.
func (tc *TypeChecker) clearCompoundNarrowing(name string) {
	for id, info := range tc.compoundNarrowingByIdentifier {
		filtered := filterOutGuard(info.guards, name)
		if len(filtered) == 0 {
			delete(tc.compoundNarrowingByIdentifier, id)
		} else {
			tc.compoundNarrowingByIdentifier[id] = compoundNarrowingInfo{guards: filtered, disp: info.disp}
		}
	}
}

// clearRefinementContext removes the matching Fact from the program-point context.
func (tc *TypeChecker) clearRefinementContext(f RefinementFact) {
	if tc.refinementCtx == nil || f.Subject == nil {
		return
	}
	for _, existing := range tc.refinementCtx.Facts() {
		if existing.Subject == nil || existing.Subject.PathKey() != f.Subject.PathKey() {
			continue
		}
		if existing.Predicate != nil && existing.Predicate.Key() == f.Predicate.Key() {
			delete(tc.refinementCtx.facts, factKey(existing.Subject, existing.Predicate))
		}
	}
}

// filterOutGuard removes a dropped guard name from NarrowingTypeGuards.
func filterOutGuard(guards []string, name string) []string {
	var out []string
	for _, g := range guards {
		if g != name {
			out = append(out, g)
		}
	}
	return out
}

// clearGuardNameFromScope walks scopes and removes guard from variable NarrowingTypeGuards.
func (tc *TypeChecker) clearGuardNameFromScope(scope *Scope, name string) {
	if scope == nil {
		return
	}
	for s := scope; s != nil; s = s.Parent {
		for id, sym := range s.Symbols {
			if sym.Kind != SymbolVariable && sym.Kind != SymbolParameter {
				continue
			}
			filtered := filterOutGuard(sym.NarrowingTypeGuards, name)
			if len(filtered) == len(sym.NarrowingTypeGuards) {
				continue
			}
			sym.NarrowingTypeGuards = filtered
			if len(filtered) == 0 {
				sym.NarrowingPredicateDisplay = ""
			}
			s.Symbols[id] = sym
		}
	}
}

// findDroppedFact returns the most recent drop of guard, if any.
func (tc *TypeChecker) findDroppedFact(guard string, argPath *AccessPath) *droppedFact {
	if tc == nil || guard == "" {
		return nil
	}
	for i := len(tc.droppedFacts) - 1; i >= 0; i-- {
		d := &tc.droppedFacts[i]
		if d.PredicateName == guard {
			return d
		}
		if guard == "Present" && (d.PredicateName == "Present" || stringsContains(d.PredicateName, "Present")) {
			return d
		}
	}
	return nil
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

// diagnosticCodeForDrop maps a dropReason to the stable refinement-invalidated-* diagnostic code.
func diagnosticCodeForDrop(r dropReason) string {
	switch r {
	case dropByAlias:
		return "refinement-invalidated-by-alias"
	case dropByCall:
		return "refinement-invalidated-by-call"
	case dropByForeign:
		return "refinement-invalidated-by-foreign"
	case dropByConcurrent:
		return "refinement-invalidated-by-concurrent"
	default:
		return "refinement-invalidated-by-write"
	}
}
