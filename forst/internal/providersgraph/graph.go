package providersgraph

import (
	"forst/internal/ast"
	"maps"
)

// Graph is the shared Providers call graph for checker, discovery JSON, and LSP (ADR-036).
type Graph struct {
	slots       map[ast.Identifier][]Slot
	direct      map[ast.Identifier]map[string]Slot
	intraEdges  []IntraPackageEdge
	moduleEdges []ModuleCallEdge
	callersOf   map[ast.Identifier][]ast.Identifier
}

// New returns an empty graph.
func New() *Graph {
	return &Graph{
		slots:     make(map[ast.Identifier][]Slot),
		direct:    make(map[ast.Identifier]map[string]Slot),
		callersOf: make(map[ast.Identifier][]ast.Identifier),
	}
}

// SetDirect records Providers introduced by use sites in fn.
func (g *Graph) SetDirect(fn ast.Identifier, slots map[string]Slot) {
	if len(slots) == 0 {
		return
	}
	copied := make(map[string]Slot, len(slots))
	maps.Copy(copied, slots)
	g.direct[fn] = copied
}

// AddIntraCall records a call edge with scope keys at the call site.
func (g *Graph) AddIntraCall(caller, callee ast.Identifier, scope ProviderScopeSnapshot) {
	g.intraEdges = append(g.intraEdges, IntraPackageEdge{
		Caller:        caller,
		Callee:        callee,
		ProviderScope: cloneProviderScope(scope),
	})
	g.callersOf[callee] = append(g.callersOf[callee], caller)
}

// AddModuleCall records a cross-package call edge.
func (g *Graph) AddModuleCall(edge ModuleCallEdge) {
	edge.ProviderScope = cloneProviderScope(edge.ProviderScope)
	g.moduleEdges = append(g.moduleEdges, edge)
}

// ComputeIntraFixedPoint runs intra-package propagation into g.slots.
func (g *Graph) ComputeIntraFixedPoint(satisfies SatisfiesProviderScope) {
	PropagateIntraPackageFixedPoint(g.slots, g.direct, g.intraEdges, satisfies)
}

// Slots returns inferred Providers for fn after fixed-point computation.
func (g *Graph) Slots(fn ast.Identifier) []Slot {
	return g.slots[fn]
}

// AllSlots returns a copy of all function Providers in the graph.
func (g *Graph) AllSlots() map[ast.Identifier][]Slot {
	out := make(map[ast.Identifier][]Slot, len(g.slots))
	for fn, slots := range g.slots {
		copied := make([]Slot, len(slots))
		copy(copied, slots)
		out[fn] = copied
	}
	return out
}

// Invalidate returns transitive intra-package callers of fn whose Providers may change
// when fn's direct use sites or Providers(g) change.
func (g *Graph) Invalidate(fn ast.Identifier) []ast.Identifier {
	seen := make(map[ast.Identifier]struct{})
	queue := []ast.Identifier{fn}
	seen[fn] = struct{}{}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, caller := range g.callersOf[cur] {
			if _, ok := seen[caller]; ok {
				continue
			}
			seen[caller] = struct{}{}
			queue = append(queue, caller)
		}
	}

	out := make([]ast.Identifier, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out
}

// ModuleGraph holds per-package Providers for cross-package fixed-point propagation.
type ModuleGraph struct {
	perPkg map[string]map[ast.Identifier][]Slot
	edges  []ModuleCallEdge
}

// NewModuleGraph clones per-package slot maps for module-level propagation.
func NewModuleGraph(perPkg map[string]map[ast.Identifier][]Slot) *ModuleGraph {
	cloned := make(map[string]map[ast.Identifier][]Slot, len(perPkg))
	for pkg, fnMap := range perPkg {
		cloned[pkg] = cloneFunctionSlots(fnMap)
	}
	return &ModuleGraph{perPkg: cloned}
}

// AddModuleCall records a cross-package edge.
func (m *ModuleGraph) AddModuleCall(edge ModuleCallEdge) {
	edge.ProviderScope = cloneProviderScope(edge.ProviderScope)
	m.edges = append(m.edges, edge)
}

// ComputeFixedPoint runs cross-package propagation.
func (m *ModuleGraph) ComputeFixedPoint(satisfies SatisfiesProviderScope) {
	PropagateModuleFixedPoint(m.perPkg, m.edges, satisfies)
}

// PerPackage returns propagated slots for one Forst package.
func (m *ModuleGraph) PerPackage(pkg string) map[ast.Identifier][]Slot {
	return m.perPkg[pkg]
}

// AllPackages returns a deep copy of all package slot maps.
func (m *ModuleGraph) AllPackages() map[string]map[ast.Identifier][]Slot {
	out := make(map[string]map[ast.Identifier][]Slot, len(m.perPkg))
	for pkg, fnMap := range m.perPkg {
		out[pkg] = cloneFunctionSlots(fnMap)
	}
	return out
}

func cloneProviderScope(src ProviderScopeSnapshot) ProviderScopeSnapshot {
	if len(src) == 0 {
		return nil
	}
	out := make(ProviderScopeSnapshot, len(src))
	maps.Copy(out, src)
	return out
}

func cloneFunctionSlots(src map[ast.Identifier][]Slot) map[ast.Identifier][]Slot {
	if len(src) == 0 {
		return make(map[ast.Identifier][]Slot)
	}
	out := make(map[ast.Identifier][]Slot, len(src))
	for fn, slots := range src {
		copied := make([]Slot, len(slots))
		copy(copied, slots)
		out[fn] = copied
	}
	return out
}
