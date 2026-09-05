package genplugin

import (
	"strings"

	"forst/internal/semantic"
)

var primitiveKinds = map[string]struct{}{
	"string": {}, "int": {}, "float": {}, "bool": {}, "bytes": {}, "void": {},
	"error": {}, "array": {}, "map": {}, "pointer": {}, "shape": {},
	"union": {}, "intersection": {}, "result": {}, "tuple": {},
	"channel": {}, "func": {}, "nominalError": {}, "alias": {},
	"goType": {}, "unknown": {},
}

// Lookup returns the snapshot type for id, or a kind-only stub when id is a primitive kind name.
func Lookup(types map[string]semantic.Type, id string) semantic.Type {
	if id == "" {
		return semantic.Type{ID: "void", Kind: "void"}
	}
	if t, ok := types[id]; ok {
		return t
	}
	if _, ok := primitiveKinds[id]; ok {
		return semantic.Type{ID: id, Kind: id}
	}
	return semantic.Type{ID: id, Kind: "unknown", Debug: id}
}

// FollowAlias walks alias/underlying links. Cycles return the last seen type.
func FollowAlias(types map[string]semantic.Type, t semantic.Type) semantic.Type {
	seen := map[string]struct{}{}
	for t.Kind == "alias" && t.Underlying != "" {
		if _, ok := seen[t.ID]; ok {
			return t
		}
		if t.ID != "" {
			seen[t.ID] = struct{}{}
		}
		t = Lookup(types, t.Underlying)
	}
	return t
}

// Resolve looks up id and follows aliases.
func Resolve(types map[string]semantic.Type, id string) semantic.Type {
	return FollowAlias(types, Lookup(types, id))
}

// ConstraintChain returns constraints on t plus those on followed aliases, in order.
func ConstraintChain(types map[string]semantic.Type, t semantic.Type) []semantic.Constraint {
	seen := map[string]struct{}{}
	var chain []semantic.Constraint
	for {
		chain = append(chain, t.Constraints...)
		if t.Kind != "alias" || t.Underlying == "" {
			break
		}
		if t.ID != "" {
			if _, ok := seen[t.ID]; ok {
				break
			}
			seen[t.ID] = struct{}{}
		}
		t = Lookup(types, t.Underlying)
	}
	return chain
}

// HasConstraintName reports whether t's own chain includes name.
func HasConstraintName(t semantic.Type, name string) bool {
	for _, c := range t.Constraints {
		if c.Name == name {
			return true
		}
	}
	return false
}

// TypeOrChainHasConstraint reports name on t or followed aliases.
func TypeOrChainHasConstraint(types map[string]semantic.Type, t semantic.Type, name string) bool {
	for _, c := range ConstraintChain(types, t) {
		if c.Name == name {
			return true
		}
	}
	return false
}

// CallView is the derived invoke view from Function.returns (SPEC: plugin-side, not snapshot fields).
type CallView struct {
	Stream  bool
	Void    bool
	Success semantic.Type
	Failure semantic.Type
}

// DerivedCall maps returns[0] to stream / Result success / unary payload.
func DerivedCall(types map[string]semantic.Type, fn semantic.Function) CallView {
	if len(fn.Returns) == 0 {
		return CallView{Void: true}
	}
	t := Resolve(types, fn.Returns[0])
	switch t.Kind {
	case "void":
		return CallView{Void: true}
	case "channel":
		elem := Lookup(types, t.Element)
		if t.Element == "" {
			elem = semantic.Type{Kind: "unknown"}
		}
		return CallView{Stream: true, Success: FollowAlias(types, elem)}
	case "result":
		if t.Success == "" && t.Failure == "" {
			return CallView{Success: semantic.Type{Kind: "unknown"}}
		}
		return CallView{
			Success: Resolve(types, t.Success),
			Failure: Resolve(types, t.Failure),
		}
	default:
		return CallView{Success: t}
	}
}

// NominalErrors returns errorSet.nominal ids, falling back to a named failure type.
func NominalErrors(types map[string]semantic.Type, fn semantic.Function) []string {
	if len(fn.ErrorSet.Nominal) > 0 {
		return append([]string(nil), fn.ErrorSet.Nominal...)
	}
	view := DerivedCall(types, fn)
	if view.Failure.Kind == "nominalError" && IsPublishedTypeID(view.Failure.ID) {
		return []string{view.Failure.ID}
	}
	return nil
}

// PackageOfTypeID returns the package prefix of a named type id.
func PackageOfTypeID(id string) string {
	if i := strings.IndexByte(id, '.'); i >= 0 {
		return id[:i]
	}
	return ""
}
