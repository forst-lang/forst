package typechecker

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// SummaryMetadataHash is a compiler-only metadata identity stub (phase 5).
// It covers package, function, format version, and sorted callee-summary keys.
func SummaryMetadataHash(pkg, fn, formatVersion string, calleeSummaryKeys []string) string {
	keys := append([]string(nil), calleeSummaryKeys...)
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(pkg)
	b.WriteByte(0)
	b.WriteString(fn)
	b.WriteByte(0)
	b.WriteString(formatVersion)
	b.WriteByte(0)
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte(0)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:16])
}

// IncrementalInvalidator tracks which callers must be revisited when a
// summary changes (phase 5 LSP/incremental hook stub).
type IncrementalInvalidator struct {
	deps  map[string]map[string]struct{} // callerKey → set of summary keys
	dirty map[string]struct{}
}

// NewIncrementalInvalidator creates an empty invalidator.
func NewIncrementalInvalidator() *IncrementalInvalidator {
	return &IncrementalInvalidator{
		deps:  make(map[string]map[string]struct{}),
		dirty: make(map[string]struct{}),
	}
}

func callerKey(pkg, fn string) string { return pkg + "\x00" + fn }
func summaryKey(pkg, fn string) string { return pkg + "\x00" + fn }

// RecordCallerDependsOn registers that caller depends on callee summary.
func (i *IncrementalInvalidator) RecordCallerDependsOn(callerPkg, callerFn, summaryPkg, summaryFn string) {
	if i == nil {
		return
	}
	if i.deps == nil {
		i.deps = make(map[string]map[string]struct{})
	}
	if i.dirty == nil {
		i.dirty = make(map[string]struct{})
	}
	ck := callerKey(callerPkg, callerFn)
	sk := summaryKey(summaryPkg, summaryFn)
	if i.deps[ck] == nil {
		i.deps[ck] = make(map[string]struct{})
	}
	i.deps[ck][sk] = struct{}{}
}

// MarkSummaryChanged dirties all callers that depend on the summary.
func (i *IncrementalInvalidator) MarkSummaryChanged(summaryPkg, summaryFn string) {
	if i == nil {
		return
	}
	if i.deps == nil {
		i.deps = make(map[string]map[string]struct{})
	}
	if i.dirty == nil {
		i.dirty = make(map[string]struct{})
	}
	sk := summaryKey(summaryPkg, summaryFn)
	for ck, deps := range i.deps {
		if _, ok := deps[sk]; ok {
			i.dirty[ck] = struct{}{}
		}
	}
}

// MustRevisitCaller reports whether the caller is dirty.
func (i *IncrementalInvalidator) MustRevisitCaller(callerPkg, callerFn string) bool {
	if i == nil {
		return false
	}
	_, ok := i.dirty[callerKey(callerPkg, callerFn)]
	return ok
}

// Clear resets the dirty set after a successful incremental pass.
func (i *IncrementalInvalidator) Clear() {
	if i == nil {
		return
	}
	i.dirty = make(map[string]struct{})
}

// GotWideningDiagnosticCode returns the stable widening diagnostic code.
func GotWideningDiagnosticCode() string {
	return "refinement-analysis-widened"
}
