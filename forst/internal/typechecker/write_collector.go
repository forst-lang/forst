package typechecker

import "forst/internal/ast"

// collectedWrite is one recorded mutation deferred until the branch/loop collector pops.
type collectedWrite struct {
	Path *AccessPath    // storage location written
	Span ast.SourceSpan // write site (invalidation diagnostic provenance)
}

// pushWriteCollector starts buffering writes so they invalidate after the whole region is analyzed.
func (tc *TypeChecker) pushWriteCollector() {
	tc.writeCollectorStack = append(tc.writeCollectorStack, nil)
}

// popWriteCollectorAndInvalidate applies buffered writes via MayClobber and drops overlapping facts.
func (tc *TypeChecker) popWriteCollectorAndInvalidate() {
	if len(tc.writeCollectorStack) == 0 {
		return
	}
	top := len(tc.writeCollectorStack) - 1
	writes := tc.writeCollectorStack[top]
	tc.writeCollectorStack = tc.writeCollectorStack[:top]
	for _, w := range writes {
		tc.invalidateOverlappingFacts(w.Path, w.Span)
	}
}

// recordBranchOrLoopWrite queues a write in the active collector (no-op if none).
func (tc *TypeChecker) recordBranchOrLoopWrite(path *AccessPath, span ast.SourceSpan) {
	if tc == nil || path == nil || len(tc.writeCollectorStack) == 0 {
		return
	}
	top := len(tc.writeCollectorStack) - 1
	tc.writeCollectorStack[top] = append(tc.writeCollectorStack[top], collectedWrite{Path: path, Span: span})
}
