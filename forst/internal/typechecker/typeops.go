package typechecker

import (
	"forst/internal/ast"
)

// InternalType is the semantic type representation used by lattice operations.
// It aliases ast.TypeNode today; a dedicated IR may replace this (see narrowing plan §9.5).
type InternalType = ast.TypeNode

// JoinAfterIfMerge implements merge-after-if for control-flow narrowing (plan §3.2, §0.3):
// at the join point following a completed if / else-if / else chain, the static type of a
// binding is the outer (pre-if) type until union types exist in the language.
//
// branchRefinements lists types inferred for the same binding along paths that narrowed it;
// outer is the binding type in the enclosing scope before the if chain. The result is always
// outer for this policy. When union/LUB types exist, consult branchRefinements to compute a
// finer join without breaking callers that already pass refinements from MergeFlowFactsAtIfJoin.
func JoinAfterIfMerge(outer InternalType, branchRefinements []InternalType) InternalType {
	_ = branchRefinements // reserved for union / LUB when InternalType can represent unions
	return outer
}

// JoinTypesOutcome describes how JoinTypes combined its inputs.
type JoinTypesOutcome int

const (
	// JoinTypesEmpty is returned when the input slice is empty.
	JoinTypesEmpty JoinTypesOutcome = iota
	// JoinTypesSingleton is a single operand (after deduplication).
	JoinTypesSingleton
	// JoinTypesAllEquivalent means every operand is structurally the same (shallow + TypeParams).
	JoinTypesAllEquivalent
	// JoinTypesHeterogeneous means operands differ; union IR is not available yet — result is first after dedupe.
	JoinTypesHeterogeneous
)

// JoinTypesDetailed classifies n-ary join: duplicates collapse for reporting, but multiple
// shallow-equal operands are JoinTypesAllEquivalent (not Singleton). Heterogeneous lists
// return the first type after DedupeTypesPreservingOrder (union IR not available yet).
func JoinTypesDetailed(types []InternalType) (InternalType, JoinTypesOutcome) {
	if len(types) == 0 {
		return InternalType{}, JoinTypesEmpty
	}
	if len(types) == 1 {
		return types[0], JoinTypesSingleton
	}
	first := types[0]
	allEq := true
	for _, t := range types[1:] {
		if !typeNodesShallowEqual(first, t) {
			allEq = false
			break
		}
	}
	if allEq {
		return first, JoinTypesAllEquivalent
	}
	deduped := DedupeTypesPreservingOrder(types)
	return deduped[0], JoinTypesHeterogeneous
}

// JoinTypes is n-ary join for type-level union (future typedef A | B). Without union IR,
// equivalent operands collapse; heterogeneous lists return (first, false).
func JoinTypes(types []InternalType) (InternalType, bool) {
	t, outcome := JoinTypesDetailed(types)
	switch outcome {
	case JoinTypesEmpty:
		return InternalType{}, false
	case JoinTypesHeterogeneous:
		return t, false
	default:
		return t, true
	}
}

// DedupeTypesPreservingOrder removes duplicate types (shallow equality) keeping first occurrence.
func DedupeTypesPreservingOrder(types []InternalType) []InternalType {
	if len(types) <= 1 {
		if len(types) == 1 {
			return []InternalType{types[0]}
		}
		return nil
	}
	var out []InternalType
	for _, t := range types {
		found := false
		for _, u := range out {
			if typeNodesShallowEqual(t, u) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, t)
		}
	}
	return out
}

func typeNodesShallowEqual(a, b InternalType) bool {
	if a.Ident != b.Ident {
		return false
	}
	if len(a.TypeParams) != len(b.TypeParams) {
		return false
	}
	for i := range a.TypeParams {
		if !typeNodesShallowEqual(a.TypeParams[i], b.TypeParams[i]) {
			return false
		}
	}
	return true
}

// MeetTypes intersects types for typedef A & B (stub: equal idents only; extend with normalization).
func MeetTypes(a, b InternalType) (InternalType, bool) {
	if typeNodesShallowEqual(a, b) {
		return a, true
	}
	return InternalType{}, false
}

// WidenToEnclosing returns the type to use after merge: the enclosing binding type (outer).
func WidenToEnclosing(outer InternalType) InternalType {
	return outer
}
