package providersgraph

import "forst/internal/ast"

// CallEdge records a function call for Providers propagation (intra- or cross-package).
type CallEdge struct {
	CallerPkg   string // set for cross-package edges after module resolution
	CallerFn    ast.Identifier
	CalleePkg   string // "" = intra-package; non-empty = Forst sibling package name
	CalleeFn    ast.Identifier
	ImportLocal string // Forst import local name; set during typecheck before CalleePkg is resolved
	Scope       ProviderScopeSnapshot
	Span        ast.SourceSpan
}

// IsCrossPackage reports whether the edge crosses Forst package boundaries.
func (e CallEdge) IsCrossPackage() bool {
	return e.CalleePkg != ""
}

// ToModuleCallEdge converts a cross-package CallEdge to ModuleCallEdge.
func (e CallEdge) ToModuleCallEdge() ModuleCallEdge {
	return ModuleCallEdge{
		CallerPkg:     e.CallerPkg,
		CallerFn:      e.CallerFn,
		TargetPkg:     e.CalleePkg,
		TargetFn:      e.CalleeFn,
		ProviderScope: e.Scope,
	}
}

// IntraEdgesFromCallEdges returns intra-package edges for fixed-point propagation.
func IntraEdgesFromCallEdges(edges []CallEdge) []IntraPackageEdge {
	var out []IntraPackageEdge
	for _, e := range edges {
		if e.IsCrossPackage() {
			continue
		}
		out = append(out, IntraPackageEdge{
			Caller:        e.CallerFn,
			Callee:        e.CalleeFn,
			ProviderScope: e.Scope,
		})
	}
	return out
}

// ModuleEdgesFromCallEdges returns cross-package edges for module fixed-point propagation.
func ModuleEdgesFromCallEdges(edges []CallEdge) []ModuleCallEdge {
	var out []ModuleCallEdge
	for _, e := range edges {
		if !e.IsCrossPackage() {
			continue
		}
		out = append(out, e.ToModuleCallEdge())
	}
	return out
}
