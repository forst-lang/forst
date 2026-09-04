package typechecker

import (
	"sort"
	"strings"

	"forst/internal/ast"
)

// AccessPaths is a set of dependency paths (Reads on a Fact).
type AccessPaths []*AccessPath

// RefinementFact is a flow-sensitive fact with dependency paths (phase 4a).
// Distinct from the minimal Fact{Subject,Predicate} in refinement_context.go;
// that type remains the RefinementContext entry. This wrapper attaches Reads + span.
type RefinementFact struct {
	Subject       *AccessPath
	Predicate     *Predicate
	Reads         AccessPaths
	EstablishedAt ast.SourceSpan
}

// PathsOverlap reports whether write path W overlaps dependency path D:
// W is a prefix of D (or equal). Writing a descendant does not overlap an ancestor dep
// (pointee write does not drop Present on the pointer slot).
func PathsOverlap(write, dep *AccessPath) bool {
	if write == nil || dep == nil {
		return false
	}
	if write.Root != dep.Root {
		return false
	}
	if len(write.Steps) > len(dep.Steps) {
		return false
	}
	for i := range write.Steps {
		if !stepsEqual(write.Steps[i], dep.Steps[i]) {
			return false
		}
	}
	return true
}

// stepsEqual compares AccessStep values (field name when AccessField).
func stepsEqual(a, b AccessStep) bool {
	if a.Kind != b.Kind {
		return false
	}
	if a.Kind == AccessField {
		return a.Field == b.Field
	}
	return true
}

// MayClobber reports whether a write to writePath may invalidate a fact that
// depends on readPath (directional overlap). IndexAny overlaps any index step.
func MayClobber(writePath, readPath *AccessPath) bool {
	if PathsOverlap(writePath, readPath) {
		return true
	}
	return pathsOverlapIndexAny(writePath, readPath)
}

// pathsOverlapIndexAny treats AccessIndexAny as overlapping any concrete or
// coarse index at the same position (phase 4e).
// pathsOverlapIndexAny treats IndexAny as overlapping any index/element write under the same prefix.
func pathsOverlapIndexAny(write, read *AccessPath) bool {
	if write == nil || read == nil || write.Root != read.Root {
		return false
	}
	n := len(write.Steps)
	if n > len(read.Steps) {
		n = len(read.Steps)
	}
	matchedPrefix := true
	sawIndex := false
	for i := 0; i < n; i++ {
		ws, rs := write.Steps[i], read.Steps[i]
		if ws.Kind == AccessIndexAny || rs.Kind == AccessIndexAny {
			sawIndex = true
			continue
		}
		if !stepsEqual(ws, rs) {
			matchedPrefix = false
			break
		}
	}
	if !matchedPrefix {
		return false
	}
	if sawIndex {
		return true
	}
	// write is a longer path under a shared prefix that includes IndexAny on write only.
	if len(write.Steps) > len(read.Steps) {
		for _, s := range write.Steps[len(read.Steps):] {
			if s.Kind == AccessIndexAny {
				return true
			}
		}
	}
	return false
}

// ExtractDepsFromAssertion collects dependency paths for an assertion IR relative
// to subjectRoot (already interned). Unanalyzable atoms depend on the whole root.
func (tc *TypeChecker) ExtractDepsFromAssertion(subjectRoot *AccessPath, a Assertion, visited map[string]struct{}) AccessPaths {
	if tc == nil || a == nil || subjectRoot == nil {
		return nil
	}
	if visited == nil {
		visited = map[string]struct{}{}
	}
	switch v := a.(type) {
	case Atom:
		return tc.extractDepsFromAtom(subjectRoot, v, visited)
	case All:
		var out AccessPaths
		for _, c := range v.Children {
			out = mergeAccessPaths(out, tc.ExtractDepsFromAssertion(subjectRoot, c, visited))
		}
		return out
	case Any:
		var out AccessPaths
		for _, c := range v.Children {
			out = mergeAccessPaths(out, tc.ExtractDepsFromAssertion(subjectRoot, c, visited))
		}
		return out
	default:
		return AccessPaths{subjectRoot}
	}
}

// extractDepsFromAtom maps one atom to dependency paths (builtins, Match shapes, named guards).
// extractDepsFromAtom resolves one IR atom to dependency paths (fail-closed to subject root).
func (tc *TypeChecker) extractDepsFromAtom(subjectRoot *AccessPath, atom Atom, visited map[string]struct{}) AccessPaths {
	name := atom.Name
	if name == "" {
		return AccessPaths{subjectRoot}
	}
	// Named type guard: union of body deps (cached).
	if tc.IsTypeGuardConstraint(name) {
		if _, seen := visited[name]; seen {
			return AccessPaths{subjectRoot}
		}
		visited[name] = struct{}{}
		return tc.depsForNamedGuard(name, subjectRoot, visited)
	}
	switch name {
	case "Present", "Nil", "Min", "Max", "Equals", "True", "False", "NotEmpty",
		ast.ValueConstraint, "Ok", "Err":
		// Builtin on the subject place itself (path already includes field if ensure was on field).
		// Collection subjects also depend on coarse [*] (phase 4e).
		out := AccessPaths{subjectRoot}
		if tc.subjectLooksLikeCollection(subjectRoot) {
			star := tc.paths.Intern(AccessPath{
				Root:  subjectRoot.Root,
				Steps: append(subjectRoot.CloneSteps(), AccessStep{Kind: AccessIndexAny}),
			})
			out = mergeAccessPaths(out, AccessPaths{star})
		}
		return out
	case "Match":
		return tc.extractDepsFromMatchShape(subjectRoot, atom)
	default:
		// Unanalyzable / unknown atom → whole root.
		rootOnly := tc.paths.Intern(AccessPath{Root: subjectRoot.Root})
		return AccessPaths{rootOnly}
	}
}

// extractDepsFromMatchShape unions field paths mentioned in Match shape arguments.
// extractDepsFromMatchShape collects field paths mentioned in a shape-match atom.
func (tc *TypeChecker) extractDepsFromMatchShape(subjectRoot *AccessPath, atom Atom) AccessPaths {
	out := AccessPaths{subjectRoot}
	for _, arg := range atom.Args {
		if arg.Shape == nil {
			continue
		}
		out = mergeAccessPaths(out, shapeFieldDeps(tc, subjectRoot, arg.Shape))
	}
	return out
}

// shapeFieldDeps walks a shape literal and returns paths for each nested field under base.
// shapeFieldDeps walks shape fields into nested AccessPaths under base.
func shapeFieldDeps(tc *TypeChecker, base *AccessPath, shape *ast.ShapeNode) AccessPaths {
	if shape == nil {
		return AccessPaths{base}
	}
	var out AccessPaths
	for name, field := range shape.Fields {
		p := tc.paths.Intern(AccessPath{
			Root:  base.Root,
			Steps: append(base.CloneSteps(), AccessStep{Kind: AccessField, Field: name}),
		})
		out = mergeAccessPaths(out, AccessPaths{p})
		if field.Shape != nil {
			out = mergeAccessPaths(out, shapeFieldDeps(tc, p, field.Shape))
		}
	}
	return out
}

// depsForNamedGuard unions body ensure deps for a type guard, cached and rebased onto subjectRoot.
// depsForNamedGuard unions dependency paths from a named type-guard body (cached, recursion-safe).
func (tc *TypeChecker) depsForNamedGuard(name string, subjectRoot *AccessPath, visited map[string]struct{}) AccessPaths {
	if cached, ok := tc.guardDepsCache[name]; ok {
		return rebasePathsToSubject(cached, subjectRoot, tc.paths)
	}
	def, ok := tc.Defs[ast.TypeIdent(name)]
	if !ok {
		rootOnly := tc.paths.Intern(AccessPath{Root: subjectRoot.Root})
		return AccessPaths{rootOnly}
	}
	var gn ast.TypeGuardNode
	switch d := def.(type) {
	case *ast.TypeGuardNode:
		gn = *d
	case ast.TypeGuardNode:
		gn = d
	default:
		rootOnly := tc.paths.Intern(AccessPath{Root: subjectRoot.Root})
		return AccessPaths{rootOnly}
	}
	subjIdent := string(gn.Subject.GetIdent())
	var out AccessPaths
	for _, node := range gn.Body {
		ens, ok := ensureStmt(node)
		if !ok {
			continue
		}
		ensPath := string(ens.Variable.Ident.ID)
		rel := relativeFieldSteps(subjIdent, ensPath)
		place := subjectRoot
		if len(rel) > 0 {
			place = tc.paths.Intern(AccessPath{Root: subjectRoot.Root, Steps: append(subjectRoot.CloneSteps(), rel...)})
		}
		ir, tt := LowerRefinementTarget(ens.Target, ens.Assertion)
		if tt != nil {
			out = mergeAccessPaths(out, AccessPaths{place})
			continue
		}
		if ir == nil {
			out = mergeAccessPaths(out, AccessPaths{place})
			continue
		}
		if len(ens.Assertion.OrChains) > 0 {
			// Union deps of all disjuncts for the compound fact.
			out = mergeAccessPaths(out, tc.ExtractDepsFromAssertion(place, ir, visited))
			continue
		}
		out = mergeAccessPaths(out, tc.ExtractDepsFromAssertion(place, ir, visited))
		// Constraint args that are other values (AllowedFor(account)) — add their paths.
		for _, c := range ens.Assertion.Constraints {
			for _, arg := range c.Args {
				if arg.Value == nil {
					continue
				}
				if vn, ok := (*arg.Value).(ast.VariableNode); ok {
					p := tc.AccessPathForVariable(&vn)
					if p != nil {
						out = mergeAccessPaths(out, AccessPaths{p})
						// Also expand known shape fields mentioned in guard? keep value root.
					}
				}
			}
		}
	}
	if tc.guardDepsCache == nil {
		tc.guardDepsCache = make(map[string]AccessPaths)
	}
	// Fail-closed: empty/unknown body → whole subject root.
	if len(out) == 0 {
		out = AccessPaths{tc.paths.Intern(AccessPath{Root: subjectRoot.Root})}
	}
	// Cache relative to a canonical root 0; rebase on use.
	tc.guardDepsCache[name] = stripRoot(out)
	return out
}

// relativeFieldSteps returns AccessSteps for ensurePath relative to guardSubject (field/[*] hops).
// relativeFieldSteps maps a dotted ensure path onto steps relative to the guard subject.
func relativeFieldSteps(guardSubject, ensurePath string) []AccessStep {
	if ensurePath == guardSubject {
		return nil
	}
	prefix := guardSubject + "."
	if !strings.HasPrefix(ensurePath, prefix) {
		return nil
	}
	var steps []AccessStep
	for _, part := range strings.Split(ensurePath[len(prefix):], ".") {
		if part == "" {
			continue
		}
		if part == "*" || part == "[*]" {
			steps = append(steps, AccessStep{Kind: AccessIndexAny})
			continue
		}
		steps = append(steps, AccessStep{Kind: AccessField, Field: part})
	}
	return steps
}

// stripRoot drops the SymbolID root, keeping only step suffixes for guard-body caching.
func stripRoot(paths AccessPaths) AccessPaths {
	out := make(AccessPaths, 0, len(paths))
	for _, p := range paths {
		if p == nil {
			continue
		}
		out = append(out, &AccessPath{Root: 0, Steps: p.CloneSteps()})
	}
	return out
}

// rebasePathsToSubject reattaches cached step suffixes onto the call-site subject path.
func rebasePathsToSubject(cached AccessPaths, subject *AccessPath, pi *PathInterner) AccessPaths {
	if subject == nil {
		return nil
	}
	out := make(AccessPaths, 0, len(cached))
	for _, p := range cached {
		if p == nil {
			continue
		}
		steps := append(subject.CloneSteps(), p.CloneSteps()...)
		out = append(out, pi.Intern(AccessPath{Root: subject.Root, Steps: steps}))
	}
	return out
}

// mergeAccessPaths unions two path sets by PathKey.
func mergeAccessPaths(a, b AccessPaths) AccessPaths {
	seen := map[string]struct{}{}
	var out AccessPaths
	add := func(p *AccessPath) {
		if p == nil {
			return
		}
		k := p.PathKey()
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		out = append(out, p)
	}
	for _, p := range a {
		add(p)
	}
	for _, p := range b {
		add(p)
	}
	return out
}

// PathKeysSorted returns stable string keys for tests.
// PathKeysSorted returns deterministic PathKey strings for fixtures/tests.
func PathKeysSorted(paths AccessPaths) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p == nil {
			continue
		}
		out = append(out, formatPathForTest(p))
	}
	sort.Strings(out)
	return out
}

func formatPathForTest(p *AccessPath) string {
	var b strings.Builder
	b.WriteString("root")
	for _, s := range p.Steps {
		switch s.Kind {
		case AccessField:
			b.WriteByte('.')
			b.WriteString(s.Field)
		case AccessIndexAny:
			b.WriteString("[*]")
		case AccessDeref:
			b.WriteString(".*")
		}
	}
	return b.String()
}

// ExtractDepsForEnsure is the public entry used by narrowing + fixtures.
func (tc *TypeChecker) ExtractDepsForEnsure(n ast.EnsureNode) AccessPaths {
	if tc.paths == nil {
		tc.paths = NewPathInterner()
	}
	subject := tc.AccessPathForVariable(&n.Variable)
	if tt, ok := n.Target.(ast.TypeTarget); ok {
		_ = tt
		return AccessPaths{subject}
	}
	if p, ok := n.Target.(*ast.TypeTarget); ok && p != nil {
		return AccessPaths{subject}
	}
	ir, _ := LowerRefinementTarget(n.Target, n.Assertion)
	if ir == nil {
		return AccessPaths{subject}
	}
	return tc.ExtractDepsFromAssertion(subject, ir, nil)
}

// ActiveFactsWithDeps returns refinement facts recorded after ensure narrowing (phase 4a store).
// ActiveFactsWithDeps returns the live fact set (with Reads) at the current program point.
func (tc *TypeChecker) ActiveFactsWithDeps() []RefinementFact {
	if tc == nil {
		return nil
	}
	return append([]RefinementFact(nil), tc.refinementFacts...)
}

// subjectLooksLikeCollection reports whether subjectRoot is bound to a slice/map/array.
// subjectLooksLikeCollection is a coarse heuristic for IndexAny dep extraction.
func (tc *TypeChecker) subjectLooksLikeCollection(subjectRoot *AccessPath) bool {
	if tc == nil || subjectRoot == nil {
		return false
	}
	for s := tc.CurrentScope(); s != nil; s = s.Parent {
		for _, sym := range s.Symbols {
			if sym.Kind != SymbolVariable || sym.ID != subjectRoot.Root {
				continue
			}
			for _, t := range sym.Types {
				if t.Ident == ast.TypeArray || t.Ident == ast.TypeMap || t.Ident == ast.TypeBytes {
					return true
				}
			}
		}
	}
	return false
}

// recordRefinementFact appends a flow-sensitive fact with deps to the active store.
// recordRefinementFact stores a newly established fact in the active fact set.
func (tc *TypeChecker) recordRefinementFact(f RefinementFact) {
	if tc == nil {
		return
	}
	tc.refinementFacts = append(tc.refinementFacts, f)
}
