package typechecker

import (
	"strings"

	"forst/internal/ast"
)

func (tc *TypeChecker) validateSidecarExportable(nodes []ast.Node) error {
	isMain := false
	for _, node := range nodes {
		if p, ok := node.(ast.PackageNode); ok {
			isMain = p.IsMainPackage()
			break
		}
	}
	if !isMain {
		return nil
	}
	for _, node := range nodes {
		fn, ok := node.(ast.FunctionNode)
		if !ok {
			continue
		}
		if fn.Receiver != nil || !ast.IsPublicExportIdent(fn.Ident.ID) {
			continue
		}
		slots := tc.FunctionUsables[fn.Ident.ID]
		if len(slots) == 0 {
			continue
		}
		roots := UsableRootIdentsFromSlots(slots)
		return diagnosticf(fn.Ident.Span, "usables-sidecar-export",
			"cannot export %s to TypeScript/sidecar: requires %s; wire at a host entry point first",
			fn.Ident.ID, strings.Join(roots, ", "))
	}
	return nil
}

func (tc *TypeChecker) finishUsablesChecking(nodes []ast.Node) error {
	tc.computeUsablesFixedPoint()

	if err := tc.validateAllCallSites(); err != nil {
		return err
	}

	for _, check := range tc.pendingWithChecks {
		tc.checkUnusedWiringKeys(check)
	}

	if err := tc.validateSidecarExportable(nodes); err != nil {
		return err
	}

	for _, node := range nodes {
		fn, ok := node.(ast.FunctionNode)
		if !ok {
			continue
		}
		if !tc.isWiringRoot(fn) {
			continue
		}
		slots := tc.FunctionUsables[fn.Ident.ID]
		if len(slots) == 0 {
			continue
		}
		names := make([]string, len(slots))
		for i, s := range slots {
			names[i] = string(s.RootIdent)
		}
		chain := obligationChain(fn.Ident.ID, slots)
		return diagnosticfRelated(fn.Ident.Span, "usables-unsatisfied", tc.usablesWiringRootRelated(fn.Ident.ID),
			"%s requires %s; not supplied at wiring root\n  required by: %s",
			fn.Ident.ID, strings.Join(names, ", "), chain)
	}
	return nil
}

func (tc *TypeChecker) usablesWiringRootRelated(rootFn ast.Identifier) []RelatedDiagnostic {
	var related []RelatedDiagnostic
	seen := make(map[ast.Identifier]struct{})
	for _, site := range tc.functionCallSites[rootFn] {
		if len(tc.FunctionUsables[site.Callee]) == 0 {
			continue
		}
		if _, ok := seen[site.Callee]; ok {
			continue
		}
		seen[site.Callee] = struct{}{}
		if sig, ok := tc.Functions[site.Callee]; ok && sig.Ident.Span.IsSet() {
			related = append(related, RelatedDiagnostic{
				Msg:  string(site.Callee) + " declares Usables requirements",
				Span: sig.Ident.Span,
			})
		}
	}
	return related
}
