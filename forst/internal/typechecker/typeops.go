package typechecker

import (
	"forst/internal/ast"
)

// InternalType is the semantic type representation used by lattice operations.
// It aliases ast.TypeNode today; a dedicated IR may replace it (see narrowing plan §9.5).
type InternalType = ast.TypeNode

// JoinAfterIfMerge combines the pre-if binding type with branch refinements at an if-chain merge point.
// When tc is nil, returns outer (conservative). Otherwise forms a union of outer and refinements,
// then collapses to outer when every union member is assignable to outer (width / subtyping).
func JoinAfterIfMerge(tc *TypeChecker, outer InternalType, branchRefinements []InternalType) InternalType {
	if len(branchRefinements) == 0 {
		return outer
	}
	if tc == nil {
		return outer
	}
	all := append([]InternalType{outer}, branchRefinements...)
	flat := flattenUnionOperandsForJoin(all)
	deduped := DedupeTypesPreservingOrder(flat)
	j, ok := JoinTypes(deduped)
	if !ok {
		return outer
	}
	if j.Ident == ast.TypeUnion {
		for _, m := range j.TypeParams {
			if !tc.IsTypeCompatible(m, outer) {
				return j
			}
		}
		return outer
	}
	if tc.IsTypeCompatible(j, outer) {
		return outer
	}
	return j
}

func flattenUnionOperandsForJoin(types []InternalType) []InternalType {
	var out []InternalType
	for _, t := range types {
		out = append(out, flattenOneForJoin(t)...)
	}
	return out
}

func flattenOneForJoin(t InternalType) []InternalType {
	if t.Ident == ast.TypeUnion && len(t.TypeParams) > 0 {
		var out []InternalType
		for _, m := range t.TypeParams {
			out = append(out, flattenOneForJoin(m)...)
		}
		return out
	}
	return []InternalType{t}
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
	// JoinTypesHeterogeneous means operands differ; result is a normalized TypeUnion (or singleton after collapse).
	JoinTypesHeterogeneous
)

// JoinTypesDetailed classifies n-ary join: duplicates collapse; heterogeneous lists become TypeUnion IR.
func JoinTypesDetailed(types []InternalType) (InternalType, JoinTypesOutcome) {
	if len(types) == 0 {
		return InternalType{}, JoinTypesEmpty
	}
	flat := flattenUnionOperandsForJoin(types)
	if len(flat) == 0 {
		return InternalType{}, JoinTypesEmpty
	}
	if len(flat) == 1 {
		return flat[0], JoinTypesSingleton
	}
	first := flat[0]
	allEq := true
	for _, t := range flat[1:] {
		if !typeNodesShallowEqual(first, t) {
			allEq = false
			break
		}
	}
	if allEq {
		return first, JoinTypesAllEquivalent
	}
	deduped := DedupeTypesPreservingOrder(flat)
	if len(deduped) == 1 {
		return deduped[0], JoinTypesSingleton
	}
	u := ast.NewUnionType(deduped...)
	return u, JoinTypesHeterogeneous
}

// JoinTypes is n-ary join for type-level union (typedef A | B and control-flow join).
func JoinTypes(types []InternalType) (InternalType, bool) {
	t, outcome := JoinTypesDetailed(types)
	switch outcome {
	case JoinTypesEmpty:
		return InternalType{}, false
	case JoinTypesHeterogeneous:
		return t, true
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

// MeetTypes intersects types for typedef A & B (structural equality or subtype GLB via MeetTypesSubtyping on use sites).
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
